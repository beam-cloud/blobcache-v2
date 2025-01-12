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
)

type PrefetchManager struct {
	ctx     context.Context
	config  BlobCacheConfig
	buffers sync.Map
}

func NewPrefetchManager(ctx context.Context, config BlobCacheConfig) *PrefetchManager {
	return &PrefetchManager{
		ctx:     ctx,
		config:  config,
		buffers: sync.Map{},
	}
}

func (pm *PrefetchManager) Start() {
	go pm.evictIdleBuffers()
}

// GetPrefetchBuffer returns an existing prefetch buffer if it exists, or nil.
func (pm *PrefetchManager) GetPrefetchBuffer(hash string, fileSize uint64) *PrefetchBuffer {
	if val, ok := pm.buffers.Load(hash); ok {
		return val.(*PrefetchBuffer)
	}

	newBuffer := NewPrefetchBuffer(hash, fileSize, pm.config.BlobFs.Prefetch.MaxBufferSizeBytes)
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
					pm.buffers.Delete(key)
				}

				return true
			})
		}
	}

}

type PrefetchBuffer struct {
	hash     string
	buffer   []byte
	lastRead time.Time
	fileSize uint64
}

func NewPrefetchBuffer(hash string, fileSize uint64, bufferSize uint64) *PrefetchBuffer {
	return &PrefetchBuffer{
		hash:     hash,
		lastRead: time.Now(),
		buffer:   make([]byte, bufferSize),
		fileSize: fileSize,
	}
}

func (pb *PrefetchBuffer) IsStale() bool {
	return time.Since(pb.lastRead) > PrefetchIdleTTL
}

func (pb *PrefetchBuffer) GetRange(offset uint64, length uint64) []byte {
	if offset+length > uint64(len(pb.buffer)) {
		return nil
	}

	go func() {
		pb.lastRead = time.Now()
	}()

	return pb.buffer[offset : offset+length]
}
