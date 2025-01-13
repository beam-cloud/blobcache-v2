package blobcache

import (
	"context"
	"sync"
	"time"
)

const (
	prefetchEvictionInterval = 30 * time.Second
	prefetchIdleTTL          = 60 * time.Second // remove stale buffers if no read in the past 60s
	preemptiveFetchThreshold = 16 * 1024 * 1024 // if the next segment is within 16MB of where we are reading, start fetching it
)

type PrefetchManager struct {
	ctx     context.Context
	config  BlobCacheConfig
	buffers sync.Map
	client  *BlobCacheClient
}

func NewPrefetchManager(ctx context.Context, config BlobCacheConfig, client *BlobCacheClient) *PrefetchManager {
	return &PrefetchManager{
		ctx:     ctx,
		config:  config,
		buffers: sync.Map{},
		client:  client,
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
		Client:      pm.client,
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

				if buffer.IsStale() {
					buffer.Stop()
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
	hash        string
	segments    map[uint64]*segment
	segmentSize uint64
	lastRead    time.Time
	fileSize    uint64
	client      *BlobCacheClient
	mu          sync.Mutex
	cond        *sync.Cond
}

type segment struct {
	data       []byte
	readLength uint64
}

type PrefetchOpts struct {
	Ctx         context.Context
	CancelFunc  context.CancelFunc
	Hash        string
	FileSize    uint64
	SegmentSize uint64
	Offset      uint64
	Client      *BlobCacheClient
}

func NewPrefetchBuffer(opts PrefetchOpts) *PrefetchBuffer {
	pb := &PrefetchBuffer{
		ctx:         opts.Ctx,
		cancelFunc:  opts.CancelFunc,
		hash:        opts.Hash,
		lastRead:    time.Now(),
		segments:    make(map[uint64]*segment),
		fileSize:    opts.FileSize,
		client:      opts.Client,
		segmentSize: opts.SegmentSize,
	}
	pb.cond = sync.NewCond(&pb.mu)
	return pb
}

func (pb *PrefetchBuffer) IsStale() bool {
	return time.Since(pb.lastRead) > prefetchIdleTTL
}

func (pb *PrefetchBuffer) fetch(offset uint64, bufferSize uint64) {
	bufferIndex := offset / bufferSize

	// Initialize internal buffer for this chunk of the content
	pb.mu.Lock()
	_, exists := pb.segments[bufferIndex]
	if exists {
		pb.mu.Unlock()
		return
	}

	s := &segment{
		data:       make([]byte, 0, bufferSize),
		readLength: 0,
	}
	pb.segments[bufferIndex] = s

	contentChan, err := pb.client.GetContentStream(pb.hash, int64(offset), int64(bufferSize))
	if err != nil {
		delete(pb.segments, bufferIndex)
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

func (pb *PrefetchBuffer) Stop() {
	pb.cancelFunc()
}

func (pb *PrefetchBuffer) GetRange(offset uint64, length uint64) []byte {
	bufferSize := pb.segmentSize
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
			// If data is not ready, wait for more data to be available
			pb.mu.Lock()
			pb.cond.Wait()
			pb.mu.Unlock()
		}
	}

	return result
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
		pb.lastRead = time.Now()

		// Calculate the relative offset within the buffer
		relativeOffset := offset - (bufferIndex * pb.segmentSize)
		availableLength := segment.readLength - relativeOffset
		readLength := min(int64(length), int64(availableLength))

		// Pre-emptively start fetching the next buffer if within the threshold
		if segment.readLength-relativeOffset <= preemptiveFetchThreshold {
			nextBufferIndex := bufferIndex + 1
			if _, nextExists := pb.segments[nextBufferIndex]; !nextExists {
				go pb.fetch(nextBufferIndex*pb.segmentSize, pb.segmentSize)
			}
		}

		return segment.data[relativeOffset : int64(relativeOffset)+int64(readLength)], true
	}

	return nil, false
}
