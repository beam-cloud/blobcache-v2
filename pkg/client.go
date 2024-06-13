package blobcache

import (
	"context"
	"errors"
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

func (c *BlobCacheClient) GetNearestHost() (*BlobCacheHost, error) {
	server := c.tailscale.GetOrCreateServer()
	client, err := server.LocalClient()
	if err != nil {
		return nil, err
	}

	hosts, err := c.discovery.FindNearbyCacheServers(context.TODO(), client)
	if err != nil {
		return nil, err
	}

	if len(hosts) == 0 {
		return nil, errors.New("no hosts found")
	}

	return hosts[0], nil
}
