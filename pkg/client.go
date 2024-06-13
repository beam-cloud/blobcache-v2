package blobcache

import (
	"fmt"

	"github.com/google/uuid"
)

type BlobCacheClient struct {
	cfg       BlobCacheConfig
	tailscale *Tailscale
	hostname  string
	discovery *DiscoveryClient
}

func NewBlobCacheClient(cfg BlobCacheConfig) (*BlobCacheClient, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheClientPrefix, uuid.New().String()[:6])
	tailscale := NewTailscale(hostname, cfg)

	return &BlobCacheClient{
		cfg:       cfg,
		hostname:  hostname,
		tailscale: tailscale,
		discovery: NewDiscoveryClient(cfg, tailscale),
	}, nil
}
