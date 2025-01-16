package blobcache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	preemptiveFetchThresholdBytes = 16 * 1024 * 1024 // if the next segment is within 16MB of where we are reading, start fetching it
)

type PrefetchManager struct {
	ctx              context.Context
	config           BlobCacheConfig
	buffers          sync.Map
	client           *BlobCacheClient
	windowIdleTTL    time.Duration
	evictionInterval time.Duration
}

func NewPrefetchManager(ctx context.Context, config BlobCacheConfig, client *BlobCacheClient) *PrefetchManager {
	return &PrefetchManager{
		ctx:              ctx,
		config:           config,
		buffers:          sync.Map{},
		client:           client,
		windowIdleTTL:    time.Duration(config.BlobFs.Prefetch.IdleTtlS) * time.Second,
		evictionInterval: time.Duration(config.BlobFs.Prefetch.EvictionIntervalS) * time.Second,
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
		case <-time.After(pm.evictionInterval):
			pm.buffers.Range(func(key, value any) bool {
				buffer := value.(*PrefetchBuffer)

				// If no reads have happened in any windows in the buffer
				// stop any fetch operations and clear the buffer so it can
				// be garbage collected
				idle := buffer.IsIdle()
				if idle {
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
	lastRead    time.Time
	windowSize  uint64
	fileSize    uint64
	client      *BlobCacheClient
	mu          sync.Mutex
	dataCond    *sync.Cond
	dataTimeout time.Duration
	windows     sync.Map
}

type window struct {
	index      int64
	data       []byte
	readLength uint64
	lastRead   time.Time
	fetching   bool
	mu         sync.Mutex
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
		windows:     sync.Map{},
		windowSize:  opts.WindowSize,
		dataTimeout: opts.DataTimeout,
		mu:          sync.Mutex{},
	}

	pb.dataCond = sync.NewCond(&pb.mu)

	return pb
}

func (pb *PrefetchBuffer) fetch(windowIndex uint64) {
	pb.mu.Lock()

	// If a window for this index is already present, don't refetch
	if _, found := pb.windows.Load(windowIndex); found {
		pb.mu.Unlock()
		return
	}

	w := &window{
		index:      int64(windowIndex),
		data:       make([]byte, 0, pb.windowSize),
		readLength: 0,
		fetching:   true,
		lastRead:   time.Now(),
		mu:         sync.Mutex{},
	}
	pb.windows.Store(windowIndex, w)

	offset := windowIndex * pb.windowSize
	contentChan, err := pb.client.GetContentStream(pb.hash, int64(offset), int64(pb.windowSize))
	if err != nil {
		pb.mu.Unlock()
		return
	}

	pb.mu.Unlock()
	for {
		select {
		case <-pb.ctx.Done():
			return
		case chunk, ok := <-contentChan:
			if !ok {
				pb.mu.Lock()

				w.mu.Lock()
				w.fetching = false
				w.lastRead = time.Now()
				w.mu.Unlock()

				// We didn't read anything for this window, so we should try again
				if w.readLength == 0 {
					pb.windows.Delete(windowIndex)
				}

				pb.dataCond.Broadcast()
				pb.mu.Unlock()
				return
			}

			w.mu.Lock()
			w.data = append(w.data, chunk...)
			w.readLength += uint64(len(chunk))
			w.lastRead = time.Now()
			pb.dataCond.Broadcast()
			w.mu.Unlock()
		}
	}
}

func (pb *PrefetchBuffer) IsIdle() bool {
	idle := true

	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.windows.Range(func(key, value any) bool {
		w := value.(*window)
		if w != nil && time.Since(w.lastRead) > pb.manager.windowIdleTTL && !w.fetching {
			return true
		} else {
			idle = false
		}

		return true
	})

	return idle
}

func (pb *PrefetchBuffer) Clear() {
	pb.cancelFunc() // Stop any fetch operations

	pb.mu.Lock()
	defer pb.mu.Unlock()

	Logger.Debugf("Evicting idle prefetch buffer - %s", pb.hash)

	// Clear all window data
	pb.windows.Range(func(key, value any) bool {
		w := value.(*window)
		if w != nil {
			w.data = nil
		}
		return true
	})
}

func (pb *PrefetchBuffer) GetRange(offset, length uint64) ([]byte, error) {
	var result []byte

	for length > 0 {
		data, ready, doneReading := pb.tryGetRange(offset, length)
		if ready {
			result = append(result, data...)
			dataLen := uint64(len(data))
			length -= dataLen
			offset += dataLen

			if doneReading {
				break
			}
		} else {
			if err := pb.waitForSignal(); err != nil {
				return nil, err
			}
		}

	}

	return result, nil
}

func (pb *PrefetchBuffer) tryGetRange(offset, length uint64) ([]byte, bool, bool) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	windowIndex := offset / pb.windowSize

	var w *window
	pb.windows.Range(func(key, value any) bool {
		win := value.(*window)
		if win != nil && win.index == int64(windowIndex) {
			w = win
			return false
		}

		return true
	})

	if w == nil {
		go pb.fetch(windowIndex)
		return nil, false, false
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	windowOffset := offset - (windowIndex * pb.windowSize)
	windowHead := (windowIndex * pb.windowSize) + w.readLength
	isLastWindow := ((windowIndex * pb.windowSize) + pb.windowSize) >= pb.fileSize

	if w.fetching && windowHead < uint64(min(int64(offset+length), int64((windowIndex+1)*pb.windowSize))) {
		return nil, false, false
	}

	if windowHead > offset {
		w.lastRead = time.Now()

		bytesAvailable := windowHead - offset
		bytesToRead := min(int64(length), int64(bytesAvailable))

		// Pre-emptively start fetching the next buffer if within the threshold
		if w.readLength-windowOffset <= preemptiveFetchThresholdBytes && !isLastWindow {
			nextWindowIndex := windowIndex + 1
			if _, found := pb.windows.Load(nextWindowIndex); !found {
				go pb.fetch(nextWindowIndex)
			}
		}

		doneReading := !w.fetching && isLastWindow
		return w.data[windowOffset : int64(windowOffset)+int64(bytesToRead)], true, doneReading
	}

	return nil, false, false
}

func (pb *PrefetchBuffer) waitForSignal() error {
	timeoutChan := time.After(pb.dataTimeout)

	for {
		select {
		case <-waitForCondition(pb.dataCond):
			return nil
		case <-timeoutChan:
			return fmt.Errorf("timeout occurred waiting for prefetch data")
		case <-pb.ctx.Done():
			return pb.ctx.Err()
		}
	}
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
