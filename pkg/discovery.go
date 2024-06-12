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
	peers     map[string]BlobCachePeer
}

func NewDiscoveryClient(cfg BlobCacheConfig, tailscale *Tailscale) *DiscoveryClient {
	return &DiscoveryClient{
		cfg:       cfg,
		tailscale: tailscale,
		peers:     make(map[string]BlobCachePeer),
	}
}

func (d *DiscoveryClient) Start(ctx context.Context) error {
	server := d.tailscale.GetOrCreateServer()
	client, err := server.LocalClient()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Duration(d.cfg.DiscoveryIntervalS) * time.Second)
	for {
		select {
		case <-ticker.C:
			err := d.findNeighbors(ctx, client)
			if err != nil {
				log.Printf("Failed to discover neighbors: %v\n", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (d *DiscoveryClient) findNeighbors(ctx context.Context, client *tailscale.LocalClient) error {
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

		if strings.Contains(peer.HostName, BlobCacheServicePrefix) {
			log.Printf("Found service @ %s\n", peer.HostName)

			wg.Add(1)

			go func(hostname string) {
				r := d.checkService(hostname)
				log.Println(hostname, ":", r)
				wg.Done()
			}(peer.HostName)

		}
	}

	wg.Wait()
	return nil
}

// checkService attempts to connect to the gRPC service and verifies its availability
func (d *DiscoveryClient) checkService(host string) bool {
	addr := fmt.Sprintf("%s:%d", host, d.cfg.Port)

	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(d.tailscale.Dial),
	)
	if err != nil {
		log.Printf("Failed to connect to gRPC service<%s>: %v", host, err)
		return false
	}
	defer conn.Close()

	c := proto.NewBlobCacheClient(conn)
	resp, err := c.GetState(context.TODO(), &proto.GetStateRequest{})
	if err != nil {
		return false
	}

	if resp.GetVersion() != BlobCacheVersion {
		return false
	}

	return true
}
