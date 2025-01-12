package blobcache

import (
	"context"
	"sync"
	"time"
)

const (
	PrefetchEvictionInterval = 30 * time.Second
	PrefetchIdleTTL          = 60 * time.Second // remove stale buffers if no read in the past 60s
	PrefetchBufferSize       = 0                // if 0, no specific limit, just store all
	PreemptiveFetchThreshold = 32 * 1024 * 1024 // 32MB
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
		Ctx:        ctx,
		CancelFunc: cancel,
		Hash:       hash,
		FileSize:   fileSize,
		BufferSize: pm.config.BlobFs.Prefetch.MaxBufferSizeBytes,
		Client:     pm.client,
	})

	pm.buffers.Store(hash, newBuffer)
	return newBuffer
}

func (pm *PrefetchManager) evictIdleBuffers() {
	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-time.After(PrefetchEvictionInterval):
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
	ctx        context.Context
	cancelFunc context.CancelFunc
	hash       string
	buffers    map[uint64]*internalBuffer
	lastRead   time.Time
	fileSize   uint64
	client     *BlobCacheClient
	mu         sync.Mutex
	cond       *sync.Cond
	bufferSize uint64
}

type internalBuffer struct {
	data       []byte
	readLength uint64
	fetching   bool
}

type PrefetchOpts struct {
	Ctx        context.Context
	CancelFunc context.CancelFunc
	Hash       string
	FileSize   uint64
	BufferSize uint64
	Offset     uint64
	Client     *BlobCacheClient
}

func NewPrefetchBuffer(opts PrefetchOpts) *PrefetchBuffer {
	pb := &PrefetchBuffer{
		ctx:        opts.Ctx,
		cancelFunc: opts.CancelFunc,
		hash:       opts.Hash,
		lastRead:   time.Now(),
		buffers:    make(map[uint64]*internalBuffer),
		fileSize:   opts.FileSize,
		client:     opts.Client,
		bufferSize: opts.BufferSize,
	}
	pb.cond = sync.NewCond(&pb.mu)
	return pb
}

func (pb *PrefetchBuffer) IsStale() bool {
	return time.Since(pb.lastRead) > PrefetchIdleTTL
}

func (pb *PrefetchBuffer) fetch(offset uint64, bufferSize uint64) {
	bufferIndex := offset / bufferSize

	// Initialize internal buffer for this chunk of the content
	pb.mu.Lock()
	_, exists := pb.buffers[bufferIndex]
	if exists {
		pb.mu.Unlock()
		return
	}

	state := &internalBuffer{
		data:       make([]byte, 0, bufferSize),
		fetching:   true,
		readLength: 0,
	}
	pb.buffers[bufferIndex] = state

	contentChan, err := pb.client.GetContentStream(pb.hash, int64(offset), int64(bufferSize))
	if err != nil {
		pb.mu.Unlock()
		// TODO: do something with this error
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
				state.fetching = false
				state.readLength = uint64(len(state.data))
				pb.cond.Broadcast()
				pb.mu.Unlock()
				return
			}

			pb.mu.Lock()
			state.data = append(state.data, chunk...)
			state.readLength = uint64(len(state.data))
			pb.cond.Broadcast()
			pb.mu.Unlock()
		}
	}
}

func (pb *PrefetchBuffer) Stop() {
	pb.cancelFunc()
}

func (pb *PrefetchBuffer) GetRange(offset uint64, length uint64) []byte {
	bufferSize := pb.bufferSize
	bufferIndex := offset / bufferSize
	bufferOffset := offset % bufferSize

	tryGetDataRange := func() ([]byte, bool) {
		pb.mu.Lock()
		defer pb.mu.Unlock()

		state, exists := pb.buffers[bufferIndex]

		// Initiate a fetch operation if the buffer does not exist
		if !exists {
			go pb.fetch(bufferIndex*bufferSize, bufferSize)
			return nil, false
		} else if state.readLength >= bufferOffset+length {
			pb.lastRead = time.Now()

			// Calculate the relative offset within the buffer
			relativeOffset := offset - (bufferIndex * bufferSize)

			// Pre-emptively start fetching the next buffer if within the threshold
			if state.readLength-relativeOffset <= PreemptiveFetchThreshold {
				nextBufferIndex := bufferIndex + 1
				if _, nextExists := pb.buffers[nextBufferIndex]; !nextExists {
					go pb.fetch(nextBufferIndex*bufferSize, bufferSize)
				}
			}

			return state.data[relativeOffset : relativeOffset+length], true
		}

		pb.cond.Wait() // Wait for more data to be available
		return nil, false
	}

	for {
		if data, ready := tryGetDataRange(); ready {
			return data
		}
	}
}
