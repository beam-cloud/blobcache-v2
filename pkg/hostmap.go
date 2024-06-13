package blobcache

import (
	"log"
	"sync"
)

func NewHostMap(onHostAdded func(*BlobCacheHost) error) *HostMap {
	return &HostMap{
		hosts:       make(map[string]*BlobCacheHost),
		mu:          sync.Mutex{},
		onHostAdded: onHostAdded,
	}
}

type HostMap struct {
	hosts       map[string]*BlobCacheHost
	mu          sync.Mutex
	onHostAdded func(*BlobCacheHost) error
}

func (hm *HostMap) Set(host *BlobCacheHost) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	_, exists := hm.hosts[host.Addr]
	if exists {
		return
	}

	hm.hosts[host.Addr] = host
	if hm.onHostAdded != nil {
		log.Println("Added new host @ ", host.Addr)
		hm.onHostAdded(host)
	}
}

func (hm *HostMap) Get(addr string) *BlobCacheHost {
	existingHost, exists := hm.hosts[addr]
	if !exists {
		return nil
	}

	return existingHost
}

func (hm *HostMap) Closest() {
}
