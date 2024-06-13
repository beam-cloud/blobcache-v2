package blobcache

import (
	"errors"
	"log"
	"sort"
	"sync"
	"time"
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
	host, exists := hm.hosts[addr]
	if !exists {
		return nil
	}

	return host
}

// Closest finds the nearest host within a given timeout
// If no hosts are found, it will error out
func (hm *HostMap) Closest(timeout time.Duration) (*BlobCacheHost, error) {
	start := time.Now()
	for {
		// If there are hosts, find the closet one
		if len(hm.hosts) > 0 {

			hm.mu.Lock()
			hosts := make([]*BlobCacheHost, 0, len(hm.hosts)) // Convert map into slice of hosts
			for _, host := range hm.hosts {
				hosts = append(hosts, host)
			}
			hm.mu.Unlock()

			// Sort by rount-trip time, return closest
			sort.Slice(hosts, func(i, j int) bool { return hosts[i].RTT < hosts[j].RTT })
			return hosts[0], nil
		}

		elapsed := time.Since(start)

		// We reached the timeout
		if elapsed > timeout {
			return nil, errors.New("no hosts found")
		}

		time.Sleep(time.Second)
	}
}
