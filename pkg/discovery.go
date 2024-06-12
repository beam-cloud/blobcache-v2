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
	tailscale     *Tailscale
	cfg           BlobCacheConfig
	peers         map[string]BlobCachePeer
	mu            sync.Mutex
	processorChan chan BlobCachePeer
}

func NewDiscoveryClient(cfg BlobCacheConfig, tailscale *Tailscale) *DiscoveryClient {
	return &DiscoveryClient{
		cfg:           cfg,
		tailscale:     tailscale,
		peers:         make(map[string]BlobCachePeer),
		processorChan: make(chan BlobCachePeer),
	}
}

func (d *DiscoveryClient) Start(ctx context.Context) error {
	server := d.tailscale.GetOrCreateServer()
	client, err := server.LocalClient()
	if err != nil {
		return err
	}

	// Start the peer processor
	go d.peerProcessor(ctx)

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

func (d *DiscoveryClient) peerProcessor(ctx context.Context) {
	for {
		select {
		case peer := <-d.processorChan:
			log.Println("Processing peer: ", peer.Addr, "RTT: ", peer.RTT)
			d.processPeer(peer)
		case <-ctx.Done():
			return
		}
	}
}

func (d *DiscoveryClient) processPeer(peer BlobCachePeer) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Naively add all peers
	d.peers[peer.Addr] = peer

	log.Println("Peers:", d.peers)
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
				defer wg.Done()
				peerData, err := d.checkService(ctx, hostname)
				if err != nil {
					log.Printf("Failed to check service %s: %v\n", hostname, err)
					return
				}

				d.processorChan <- *peerData
			}(peer.DNSName)
		}
	}

	wg.Wait()
	return nil
}

// checkService attempts to connect to the gRPC service and verifies its availability
func (d *DiscoveryClient) checkService(ctx context.Context, host string) (*BlobCachePeer, error) {
	addr := fmt.Sprintf("%s:%d", host, d.cfg.Port)
	peerData := BlobCachePeer{
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
