package blobcache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"tailscale.com/client/tailscale"
)

type BlobCacheClient struct {
	cfg       BlobCacheConfig
	tailscale *Tailscale
	hostname  string
	discovery *DiscoveryClient
	client    *tailscale.LocalClient
}

func NewBlobCacheClient(cfg BlobCacheConfig) (*BlobCacheClient, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheClientPrefix, uuid.New().String()[:6])
	tailscale := NewTailscale(hostname, cfg)

	server := tailscale.GetOrCreateServer()
	client, err := server.LocalClient()
	if err != nil {
		return nil, err
	}

	return &BlobCacheClient{
		cfg:       cfg,
		hostname:  hostname,
		tailscale: tailscale,
		discovery: NewDiscoveryClient(cfg, tailscale),
		client:    client,
	}, nil
}

func (c *BlobCacheClient) GetNearestHost() (*BlobCacheHost, error) {
	for {
		hosts, err := c.discovery.FindNearbyCacheServers(context.TODO(), c.client)
		if err != nil {
			return nil, err
		}

		if len(hosts) > 0 {
			log.Println("HOST: ", hosts[0])
		}

		time.Sleep(time.Second)

		// if len(hosts) == 0 {
		// 	return nil, errors.New("no hosts found")
		// }
	}

	return nil, nil
}
