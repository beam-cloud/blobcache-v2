package blobcache

// Based on https://github.com/tysonmote/rendezvous and adapted to use a HostMap instead of a list of strings

import (
	"bytes"
	"hash"
	"hash/crc32"
	"sort"
	"unsafe"
)

var crc32Table = crc32.MakeTable(crc32.Castagnoli)

type RendezvousHasher struct {
	hosts  hostScores
	hasher hash.Hash32
}

type hostScore struct {
	hostId []byte
	host   *BlobCacheHost
	score  uint32
}

func NewRendezvousHasher() *RendezvousHasher {
	hasher := &RendezvousHasher{
		hasher: crc32.New(crc32Table),
	}
	return hasher
}

// Add adds additional hosts
func (h *RendezvousHasher) Add(hosts ...*BlobCacheHost) {
	for _, host := range hosts {
		h.hosts = append(h.hosts, hostScore{host: host, hostId: []byte(host.HostId)})
	}
}

// Remove removes a host
func (h *RendezvousHasher) Remove(host *BlobCacheHost) {
	for i, ns := range h.hosts {
		if ns.host.HostId == host.HostId {
			h.hosts = append(h.hosts[:i], h.hosts[i+1:]...)
			return
		}
	}
}

// Get returns the host with the highest score for the given key. If this RendezvousHasher
// has no hosts, an empty string is returned.
func (h *RendezvousHasher) Get(key string) *BlobCacheHost {
	var maxScore uint32
	var maxHost *BlobCacheHost = nil
	var maxHostId []byte

	keyBytes := unsafeBytes(key)

	for _, host := range h.hosts {
		score := h.hash(host.hostId, keyBytes)
		if score > maxScore || (score == maxScore && bytes.Compare(host.hostId, maxHostId) < 0) {
			maxScore = score
			maxHost = host.host
			maxHostId = host.hostId
		}
	}

	return maxHost
}

// GetN returns no more than n hosts for the given key, ordered by descending
// score. GetN is not goroutine-safe.
func (h *RendezvousHasher) GetN(n int, key string) []*BlobCacheHost {
	keyBytes := unsafeBytes(key)
	for i := range h.hosts {
		h.hosts[i].score = h.hash(h.hosts[i].hostId, keyBytes)
	}
	sort.Sort(&h.hosts)

	if n > len(h.hosts) {
		n = len(h.hosts)
	}

	hosts := make([]*BlobCacheHost, n)
	for i := range hosts {
		hosts[i] = h.hosts[i].host
	}
	return hosts
}

type hostScores []hostScore

func (s *hostScores) Len() int      { return len(*s) }
func (s *hostScores) Swap(i, j int) { (*s)[i], (*s)[j] = (*s)[j], (*s)[i] }
func (s *hostScores) Less(i, j int) bool {
	if (*s)[i].score == (*s)[j].score {
		return bytes.Compare((*s)[i].hostId, (*s)[j].hostId) < 0
	}
	return (*s)[j].score < (*s)[i].score // Descending
}

func (h *RendezvousHasher) hash(hostId, key []byte) uint32 {
	h.hasher.Reset()
	h.hasher.Write(key)
	h.hasher.Write(hostId)
	return h.hasher.Sum32()
}

func unsafeBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
