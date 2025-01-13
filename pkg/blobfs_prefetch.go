package blobcache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	prefetchEvictionInterval      = 5 * time.Second
	prefetchSegmentIdleTTL        = 5 * time.Second  // remove stale segments if no reads in the past 30s
	preemptiveFetchThresholdBytes = 16 * 1024 * 1024 // if the next segment is within 16MB of where we are reading, start fetching it
)

type PrefetchManager struct {
	ctx                      context.Context
	config                   BlobCacheConfig
	buffers                  sync.Map
	client                   *BlobCacheClient
	currentPrefetchSizeBytes uint64
}

func NewPrefetchManager(ctx context.Context, config BlobCacheConfig, client *BlobCacheClient) *PrefetchManager {
	return &PrefetchManager{
		ctx:                      ctx,
		config:                   config,
		buffers:                  sync.Map{},
		client:                   client,
		currentPrefetchSizeBytes: 0,
	}
}

func (pm *PrefetchManager) Start() {
	go pm.evictIdleBuffers()
}

// GetPrefetchBuffer returns an existing prefetch buffer if it exists, or nil
func (pm *PrefetchManager) GetPrefetchBuffer(hash string, fileSize uint64) *PrefetchBuffer {
	if val, ok := pm.buffers.Load(hash); ok {
		return val.(*PrefetchBuffer)
	}

	ctx, cancel := context.WithCancel(pm.ctx)
	newBuffer := NewPrefetchBuffer(PrefetchOpts{
		Ctx:         ctx,
		CancelFunc:  cancel,
		Hash:        hash,
		FileSize:    fileSize,
		WindowSize:  pm.config.BlobFs.Prefetch.WindowSizeBytes,
		DataTimeout: time.Second * time.Duration(pm.config.BlobFs.Prefetch.DataTimeoutS),
		Client:      pm.client,
		Manager:     pm,
	})

	pm.buffers.Store(hash, newBuffer)
	return newBuffer
}

func (pm *PrefetchManager) evictIdleBuffers() {
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-time.After(prefetchEvictionInterval):
			pm.buffers.Range(func(key, value any) bool {
				buffer := value.(*PrefetchBuffer)

				// If no reads have happened in any segments in the buffer
				// stop any fetch operations and clear the buffer so it can
				// be garbage collected
				unused := buffer.evictIdle()
				if unused {
					buffer.Clear()
					pm.buffers.Delete(key)
				}

				return true
			})
		}
	}
}

type PrefetchBuffer struct {
	ctx         context.Context
	cancelFunc  context.CancelFunc
	manager     *PrefetchManager
	hash        string
	windows     map[uint64]*window
	windowSize  uint64
	lastRead    time.Time
	fileSize    uint64
	client      *BlobCacheClient
	mu          sync.Mutex
	dataCond    *sync.Cond
	dataTimeout time.Duration
}

type window struct {
	index      uint64
	data       []byte
	readLength uint64
	lastRead   time.Time
	fetching   bool
}

type PrefetchOpts struct {
	Ctx         context.Context
	CancelFunc  context.CancelFunc
	Hash        string
	FileSize    uint64
	WindowSize  uint64
	Offset      uint64
	Client      *BlobCacheClient
	DataTimeout time.Duration
	Manager     *PrefetchManager
}

func NewPrefetchBuffer(opts PrefetchOpts) *PrefetchBuffer {
	pb := &PrefetchBuffer{
		ctx:         opts.Ctx,
		cancelFunc:  opts.CancelFunc,
		hash:        opts.Hash,
		manager:     opts.Manager,
		lastRead:    time.Now(),
		fileSize:    opts.FileSize,
		client:      opts.Client,
		windows:     make(map[uint64]*window),
		windowSize:  opts.WindowSize,
		dataTimeout: opts.DataTimeout,
		mu:          sync.Mutex{},
	}
	pb.dataCond = sync.NewCond(&pb.mu)
	return pb
}

func (pb *PrefetchBuffer) fetch(offset uint64, bufferSize uint64) {
	bufferIndex := offset / bufferSize

	pb.mu.Lock()
	if _, exists := pb.windows[bufferIndex]; exists {
		pb.mu.Unlock()
		return
	}

	w := &window{
		index:      bufferIndex,
		data:       make([]byte, 0, bufferSize),
		readLength: 0,
		lastRead:   time.Now(),
		fetching:   true,
	}
	pb.windows[bufferIndex] = w
	pb.mu.Unlock()

	contentChan, err := pb.client.GetContentStream(pb.hash, int64(offset), int64(bufferSize))
	if err != nil {
		pb.mu.Lock()
		delete(pb.windows, bufferIndex)
		pb.mu.Unlock()
		return
	}

	for {
		select {
		case <-pb.ctx.Done():
			return
		case chunk, ok := <-contentChan:
			if !ok {
				pb.mu.Lock()
				w.fetching = false
				w.lastRead = time.Now()
				pb.dataCond.Broadcast()
				pb.mu.Unlock()
				return
			}

			pb.mu.Lock()
			w.data = append(w.data, chunk...)
			w.readLength += uint64(len(chunk))
			w.lastRead = time.Now()
			pb.dataCond.Broadcast()
			pb.mu.Unlock()
		}
	}
}

func (pb *PrefetchBuffer) evictIdle() bool {
	unused := true
	var indicesToDelete []uint64

	pb.mu.Lock()
	for index, window := range pb.windows {
		if time.Since(window.lastRead) > prefetchSegmentIdleTTL && !window.fetching {
			indicesToDelete = append(indicesToDelete, index)
		} else {
			unused = false
		}
	}
	pb.mu.Unlock()

	for _, index := range indicesToDelete {
		pb.mu.Lock()
		Logger.Infof("Evicting segment %s-%d", pb.hash, index)
		window := pb.windows[index]
		window.data = nil
		delete(pb.windows, index)
		pb.mu.Unlock()
	}

	return unused
}

func (pb *PrefetchBuffer) Clear() {
	pb.cancelFunc() // Stop any fetch operations

	pb.mu.Lock()
	defer pb.mu.Unlock()

	// Clear all window data
	for _, window := range pb.windows {
		window.data = nil
	}

	// Reinitialize the map to clear all entries
	pb.windows = make(map[uint64]*window)
}

func (pb *PrefetchBuffer) GetRange(offset, length uint64) ([]byte, error) {
	bufferSize := pb.windowSize
	bufferIndex := offset / bufferSize
	bufferOffset := offset % bufferSize

	var result []byte

	for length > 0 {
		data, ready := pb.tryGetRange(bufferIndex, bufferOffset, offset, length)
		if ready {
			result = append(result, data...)
			dataLen := uint64(len(data))
			length -= dataLen
			offset += dataLen
			bufferIndex = offset / bufferSize
			bufferOffset = offset % bufferSize
		} else {
			Logger.Infof("Waiting for prefetch signal")
			if err := pb.waitForSignal(); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func (pb *PrefetchBuffer) waitForSignal() error {
	timeoutChan := time.After(pb.dataTimeout)

	for {
		select {
		case <-waitForCondition(pb.dataCond):
			Logger.Infof("Prefetch data ready")
			return nil
		case <-timeoutChan:
			Logger.Infof("Timeout occurred waiting for prefetch data")
			return fmt.Errorf("timeout occurred waiting for prefetch data")
		case <-pb.ctx.Done():
			return fmt.Errorf("context canceled")
		}
	}
}

func (pb *PrefetchBuffer) tryGetRange(bufferIndex, bufferOffset, offset, length uint64) ([]byte, bool) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	window, exists := pb.windows[bufferIndex]

	// Initiate a fetch operation if the buffer does not exist
	if !exists {
		Logger.Infof("Fetching segment %s-%d", pb.hash, bufferIndex)
		go pb.fetch(bufferIndex*pb.windowSize, pb.windowSize)
		return nil, false
	} else if window.readLength > bufferOffset {
		window.lastRead = time.Now()

		// Calculate the relative offset within the buffer
		relativeOffset := offset - (bufferIndex * pb.windowSize)
		availableLength := window.readLength - relativeOffset
		readLength := min(int64(length), int64(availableLength))

		// Pre-emptively start fetching the next buffer if within the threshold
		if window.readLength-relativeOffset <= preemptiveFetchThresholdBytes {
			nextBufferIndex := bufferIndex + 1
			if _, nextExists := pb.windows[nextBufferIndex]; !nextExists {
				go pb.fetch(nextBufferIndex*pb.windowSize, pb.windowSize)
			}
		}

		return window.data[relativeOffset : int64(relativeOffset)+int64(readLength)], true
	}

	return nil, false
}

func waitForCondition(cond *sync.Cond) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		cond.L.Lock()
		cond.Wait()
		cond.L.Unlock()
		close(ch)
	}()
	return ch
}
