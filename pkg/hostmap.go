package blobcache

import (
	"errors"
	"sort"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
)

func NewHostMap(cfg BlobCacheConfig, onHostAdded func(*BlobCacheHost) error) *HostMap {
	return &HostMap{
		hosts:       make(map[string]*BlobCacheHost),
		mu:          sync.Mutex{},
		onHostAdded: onHostAdded,
		cfg:         cfg,
	}
}

type HostMap struct {
	hosts       map[string]*BlobCacheHost
	mu          sync.Mutex
	onHostAdded func(*BlobCacheHost) error
	cfg         BlobCacheConfig
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
		Logger.Infof("Added new host @ %s (PrivateAddr=%s, RTT=%s)", host.Addr, host.PrivateAddr, host.RTT)
		hm.onHostAdded(host)
	}
}

func (hm *HostMap) Remove(host *BlobCacheHost) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	_, exists := hm.hosts[host.Addr]
	if !exists {
		return
	}

	delete(hm.hosts, host.Addr)
}

func (hm *HostMap) Members() mapset.Set[string] {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	set := mapset.NewSet[string]()
	for addr := range hm.hosts {
		set.Add(hm.hosts[addr].Addr)
	}

	return set
}

func (hm *HostMap) Get(addr string) *BlobCacheHost {
	hm.mu.Lock()
	defer hm.mu.Unlock()

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

// ClosestWithCapacity finds the nearest host with available storage capacity within a given timeout
// If no hosts are found, it will error out
func (hm *HostMap) ClosestWithCapacity(timeout time.Duration) (*BlobCacheHost, error) {
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

			// Sort by rount-trip time and capacity usage
			sort.Slice(hosts, func(i, j int) bool {
				if hosts[i].RTT != hosts[j].RTT {
					return hosts[i].RTT < hosts[j].RTT
				}
				return hosts[i].CapacityUsagePct < hosts[j].CapacityUsagePct
			})

			if len(hosts) == 0 {
				return nil, errors.New("no hosts found")
			}

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
