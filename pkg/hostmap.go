package blobcache

import "sync"

func NewHostMap() *HostMap {
	return &HostMap{
		hosts: make(map[string]*BlobCacheHost),
		mu:    sync.Mutex{},
	}
}

type HostMap struct {
	hosts map[string]*BlobCacheHost
	mu    sync.Mutex
}

func (hm *HostMap) Set(host *BlobCacheHost) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	_, exists := hm.hosts[host.Addr]
	if exists {
		return
	}

	hm.hosts[host.Addr] = host
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
