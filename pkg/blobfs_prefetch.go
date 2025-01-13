package blobcache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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
	totalPrefetchSizeBytes   uint64
}

func NewPrefetchManager(ctx context.Context, config BlobCacheConfig, client *BlobCacheClient) *PrefetchManager {
	return &PrefetchManager{
		ctx:                      ctx,
		config:                   config,
		buffers:                  sync.Map{},
		client:                   client,
		currentPrefetchSizeBytes: 0,
		totalPrefetchSizeBytes:   config.BlobFs.Prefetch.TotalPrefetchSizeBytes,
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
		SegmentSize: pm.config.BlobFs.Prefetch.SegmentSizeBytes,
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

func (pm *PrefetchManager) incrementPrefetchSize(size uint64) bool {
	newTotal := atomic.AddUint64(&pm.currentPrefetchSizeBytes, size)
	if newTotal > pm.totalPrefetchSizeBytes {
		atomic.AddUint64(&pm.currentPrefetchSizeBytes, ^uint64(size-1))
		return false
	}

	return true
}

func (pm *PrefetchManager) decrementPrefetchSize(size uint64) {
	atomic.AddUint64(&pm.currentPrefetchSizeBytes, ^uint64(size-1))
}

type PrefetchBuffer struct {
	ctx         context.Context
	cancelFunc  context.CancelFunc
	manager     *PrefetchManager
	hash        string
	segments    map[uint64]*segment
	segmentSize uint64
	lastRead    time.Time
	fileSize    uint64
	client      *BlobCacheClient
	mu          sync.Mutex
	cond        *sync.Cond
	dataTimeout time.Duration
}

type segment struct {
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
	SegmentSize uint64
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
		segments:    make(map[uint64]*segment),
		segmentSize: opts.SegmentSize,
		dataTimeout: opts.DataTimeout,
	}
	pb.cond = sync.NewCond(&pb.mu)
	return pb
}

func (pb *PrefetchBuffer) fetch(offset uint64, bufferSize uint64) {
	bufferIndex := offset / bufferSize

	pb.mu.Lock()
	if !pb.manager.incrementPrefetchSize(bufferSize) {
		pb.mu.Unlock()
		return
	}

	if _, exists := pb.segments[bufferIndex]; exists {
		pb.manager.decrementPrefetchSize(bufferSize)
		pb.mu.Unlock()
		return
	}

	s := &segment{
		index:      bufferIndex,
		data:       make([]byte, 0, bufferSize),
		readLength: 0,
		lastRead:   time.Now(),
		fetching:   true,
	}
	pb.segments[bufferIndex] = s
	pb.mu.Unlock()

	contentChan, err := pb.client.GetContentStream(pb.hash, int64(offset), int64(bufferSize))
	if err != nil {
		pb.mu.Lock()
		delete(pb.segments, bufferIndex)
		pb.manager.decrementPrefetchSize(bufferSize)
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
				s.fetching = false
				s.lastRead = time.Now()
				pb.cond.Broadcast()
				pb.mu.Unlock()
				return
			}

			pb.mu.Lock()
			s.data = append(s.data, chunk...)
			s.readLength += uint64(len(chunk))
			pb.cond.Broadcast()
			pb.mu.Unlock()
		}
	}
}

func (pb *PrefetchBuffer) evictIdle() bool {
	unused := true
	var indicesToDelete []uint64

	pb.mu.Lock()
	for index, segment := range pb.segments {
		if time.Since(segment.lastRead) > prefetchSegmentIdleTTL && !segment.fetching {
			indicesToDelete = append(indicesToDelete, index)
		} else {
			unused = false
		}
	}
	pb.mu.Unlock()

	for _, index := range indicesToDelete {
		pb.mu.Lock()
		Logger.Infof("Evicting segment %s-%d", pb.hash, index)
		segment := pb.segments[index]
		segmentSize := uint64(len(segment.data))
		segment.data = nil
		delete(pb.segments, index)
		pb.manager.decrementPrefetchSize(segmentSize)
		pb.cond.Broadcast()
		pb.mu.Unlock()
	}

	return unused
}

func (pb *PrefetchBuffer) Clear() {
	pb.cancelFunc() // Stop any fetch operations

	pb.mu.Lock()
	defer pb.mu.Unlock()

	// Clear all segment data
	for _, segment := range pb.segments {
		segment.data = nil
	}

	// Reinitialize the map to clear all entries
	pb.segments = make(map[uint64]*segment)
}

func (pb *PrefetchBuffer) GetRange(offset uint64, length uint64) ([]byte, error) {
	bufferSize := pb.segmentSize
	bufferIndex := offset / bufferSize
	bufferOffset := offset % bufferSize

	var result []byte
	timeoutChan := time.After(pb.dataTimeout)

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
			select {
			case <-timeoutChan:
				return nil, fmt.Errorf("timeout occurred waiting for prefetch data")
			default:
				// If data is not ready, wait for more data to be available
				pb.mu.Lock()
				pb.cond.Wait()
				pb.mu.Unlock()
			}
		}
	}

	return result, nil
}

func (pb *PrefetchBuffer) tryGetRange(bufferIndex, bufferOffset, offset, length uint64) ([]byte, bool) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	segment, exists := pb.segments[bufferIndex]

	// Initiate a fetch operation if the buffer does not exist
	if !exists {
		go pb.fetch(bufferIndex*pb.segmentSize, pb.segmentSize)
		return nil, false
	} else if segment.readLength > bufferOffset {
		segment.lastRead = time.Now()

		// Calculate the relative offset within the buffer
		relativeOffset := offset - (bufferIndex * pb.segmentSize)
		availableLength := segment.readLength - relativeOffset
		readLength := min(int64(length), int64(availableLength))

		// Pre-emptively start fetching the next buffer if within the threshold
		if segment.readLength-relativeOffset <= preemptiveFetchThresholdBytes {
			nextBufferIndex := bufferIndex + 1
			if _, nextExists := pb.segments[nextBufferIndex]; !nextExists {
				go pb.fetch(nextBufferIndex*pb.segmentSize, pb.segmentSize)
			}
		}

		return segment.data[relativeOffset : int64(relativeOffset)+int64(readLength)], true
	}

	return nil, false
}
