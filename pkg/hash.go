package blobcache

// Vendored from github.com/tysonmote/rendezvous

import (
	"bytes"
	"hash"
	"hash/crc32"
	"sort"
	"unsafe"
)

var crc32Table = crc32.MakeTable(crc32.Castagnoli)

type RendezvousHasher struct {
	nodes   nodeScores
	hasher  hash.Hash32
	hostMap *HostMap
}

type nodeScore struct {
	node  []byte
	score uint32
}

// New returns a new Hash ready for use with the given nodes.
func NewRendezvousHasher(hostMap *HostMap) *RendezvousHasher {
	hasher := &RendezvousHasher{
		hasher:  crc32.New(crc32Table),
		hostMap: hostMap,
	}

	for _, h := range hostMap.hosts {
		hasher.Add(h.Host)
	}

	return hasher
}

// Add adds additional hosts to the Hash.
func (h *RendezvousHasher) Add(nodes ...string) {
	for _, node := range nodes {
		h.nodes = append(h.nodes, nodeScore{node: []byte(node)})
	}
}

// Remove removes a node from the Hash, if it exists
func (h *RendezvousHasher) Remove(node string) {
	nodeBytes := []byte(node)
	for i, ns := range h.nodes {
		if bytes.Equal(ns.node, nodeBytes) {
			h.nodes = append(h.nodes[:i], h.nodes[i+1:]...)
			return
		}
	}
}

// Get returns the node with the highest score for the given key. If this Hash
// has no nodes, an empty string is returned.
func (h *RendezvousHasher) Get(key string) string {
	var maxScore uint32
	var maxNode []byte

	keyBytes := unsafeBytes(key)

	for _, node := range h.nodes {
		score := h.hash(node.node, keyBytes)
		if score > maxScore || (score == maxScore && bytes.Compare(node.node, maxNode) < 0) {
			maxScore = score
			maxNode = node.node
		}
	}

	return string(maxNode)
}

// GetN returns no more than n nodes for the given key, ordered by descending
// score. GetN is not goroutine-safe.
func (h *RendezvousHasher) GetN(n int, key string) []string {
	keyBytes := unsafeBytes(key)
	for i := 0; i < len(h.nodes); i++ {
		h.nodes[i].score = h.hash(h.nodes[i].node, keyBytes)
	}
	sort.Sort(&h.nodes)

	if n > len(h.nodes) {
		n = len(h.nodes)
	}

	nodes := make([]string, n)
	for i := range nodes {
		nodes[i] = string(h.nodes[i].node)
	}

	return nodes
}

type nodeScores []nodeScore

func (s *nodeScores) Len() int      { return len(*s) }
func (s *nodeScores) Swap(i, j int) { (*s)[i], (*s)[j] = (*s)[j], (*s)[i] }
func (s *nodeScores) Less(i, j int) bool {
	if (*s)[i].score == (*s)[j].score {
		return bytes.Compare((*s)[i].node, (*s)[j].node) < 0
	}
	return (*s)[j].score < (*s)[i].score // Descending
}

func (h *RendezvousHasher) hash(node, key []byte) uint32 {
	h.hasher.Reset()
	h.hasher.Write(key)
	h.hasher.Write(node)
	return h.hasher.Sum32()
}

func unsafeBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
