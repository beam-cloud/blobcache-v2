package blobcache

import (
	"context"
	"sync"
	"time"
)

const (
	PrefetchEvictionInterval = 30 * time.Second
	PrefetchIdleTTL          = 60 * time.Second // e.g. remove buffer if no read in the past 60s
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
func (pm *PrefetchManager) GetPrefetchBuffer(hash string) *PrefetchBuffer {
	if val, ok := pm.buffers.Load(hash); ok {
		return val.(*PrefetchBuffer)
	}

	return nil
}

func (pm *PrefetchManager) evictIdleBuffers() {

}

type PrefetchBuffer struct {
	hash     string
	buffer   []byte
	lastRead time.Time
}
