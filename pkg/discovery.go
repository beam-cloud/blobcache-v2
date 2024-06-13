package blobcache

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	proto "github.com/beam-cloud/blobcache/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"tailscale.com/client/tailscale"
)

type DiscoveryClient struct {
	tailscale *Tailscale
	cfg       BlobCacheConfig
	hosts     map[string]BlobCacheHost
	mu        sync.Mutex
}

func NewDiscoveryClient(cfg BlobCacheConfig, tailscale *Tailscale) *DiscoveryClient {
	return &DiscoveryClient{
		cfg:       cfg,
		tailscale: tailscale,
		hosts:     make(map[string]BlobCacheHost),
	}
}

// Used by blobcache servers to discover their closest peers
func (d *DiscoveryClient) StartInBackground(ctx context.Context) error {
	server := d.tailscale.GetOrCreateServer()
	client, err := server.LocalClient()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(d.cfg.DiscoveryIntervalS) * time.Second)
	for {
		select {
		case <-ticker.C:
			err := d.FindNearbyCacheServers(ctx, client)
			if err != nil {
				log.Printf("Failed to discover neighbors: %v\n", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (d *DiscoveryClient) FindNearbyCacheServers(ctx context.Context, client *tailscale.LocalClient) error {
	status, err := client.Status(ctx)
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	// Iterate through the peers to find any matching blobcache services
	for _, peer := range status.Peer {
		if !peer.Online {
			continue
		}

		if strings.Contains(peer.HostName, BlobCacheHostPrefix) {
			log.Printf("Found service @ %s\n", peer.HostName)

			wg.Add(1)

			go func(hostname string) {
				defer wg.Done()

				hostState, err := d.GetHostState(ctx, hostname)
				if err != nil {
					return
				}

				log.Printf("hostState %v\n", hostState)

			}(peer.DNSName)
		}
	}

	wg.Wait()
	return nil
}

// checkService attempts to connect to the gRPC service and verifies its availability
func (d *DiscoveryClient) GetHostState(ctx context.Context, host string) (*BlobCacheHost, error) {
	addr := fmt.Sprintf("%s:%d", host, d.cfg.Port)
	peerData := BlobCacheHost{
		Addr: host,
		RTT:  0,
	}

	var dialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(d.tailscale.Dial),
	}

	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	startTime := time.Now()
	c := proto.NewBlobCacheClient(conn)
	resp, err := c.GetState(ctx, &proto.GetStateRequest{})
	if err != nil {
		return nil, err
	}
	peerData.RTT = time.Since(startTime)

	if resp.GetVersion() != BlobCacheVersion {
		return nil, fmt.Errorf("version mismatch: %s != %s", resp.GetVersion(), BlobCacheVersion)
	}

	return &peerData, nil
}
