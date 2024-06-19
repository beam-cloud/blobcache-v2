package blobcache

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"tailscale.com/client/tailscale"
)

type DiscoveryClient struct {
	tailscale *Tailscale
	cfg       BlobCacheConfig
	hostMap   *HostMap
	mu        sync.Mutex
}

func NewDiscoveryClient(cfg BlobCacheConfig, tailscale *Tailscale, hostMap *HostMap) *DiscoveryClient {
	return &DiscoveryClient{
		cfg:       cfg,
		tailscale: tailscale,
		hostMap:   hostMap,
	}
}

func (d *DiscoveryClient) updateHostMap(newHosts []*BlobCacheHost) {
	for _, h := range newHosts {
		d.hostMap.Set(h)
	}
}

// Used by blobcache servers to discover their closest peers
func (d *DiscoveryClient) StartInBackground(ctx context.Context) error {
	server := d.tailscale.GetOrCreateServer()
	client, err := server.LocalClient()
	if err != nil {
		return err
	}

	hosts, err := d.FindNearbyHosts(ctx, client)
	if err == nil {
		d.updateHostMap(hosts)
	}

	ticker := time.NewTicker(time.Duration(d.cfg.DiscoveryIntervalS) * time.Second)
	for {
		select {
		case <-ticker.C:
			hosts, err := d.FindNearbyHosts(ctx, client)
			if err != nil {
				continue
			}

			d.updateHostMap(hosts)
		case <-ctx.Done():
			return nil
		}
	}
}

func (d *DiscoveryClient) FindNearbyHosts(ctx context.Context, client *tailscale.LocalClient) ([]*BlobCacheHost, error) {
	status, err := client.Status(ctx)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	hosts := []*BlobCacheHost{}

	// Iterate through the peers to find any matching blobcache services
	for _, peer := range status.Peer {
		if !peer.Online {
			continue
		}

		if strings.Contains(peer.HostName, BlobCacheHostPrefix) {
			wg.Add(1)

			go func(hostname string) {
				defer wg.Done()

				hostname = hostname[:len(hostname)-1] // Strip the last period
				addr := fmt.Sprintf("%s:%d", hostname, d.cfg.Port)
				host, err := d.GetHostState(ctx, addr)
				if err != nil {
					return
				}

				d.mu.Lock()
				defer d.mu.Unlock()
				hosts = append(hosts, host)

			}(peer.DNSName)
		}
	}

	wg.Wait()
	return hosts, nil
}

// checkService attempts to connect to the gRPC service and verifies its availability
func (d *DiscoveryClient) GetHostState(ctx context.Context, addr string) (*BlobCacheHost, error) {
	host := BlobCacheHost{
		Addr: addr,
		RTT:  0,
	}

	var dialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(d.tailscale.DialWithTimeout),
	}

	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Query host state to figure out what the round-trip times might look like
	startTime := time.Now()
	c := proto.NewBlobCacheClient(conn)
	resp, err := c.GetState(ctx, &proto.GetStateRequest{})
	if err != nil {
		return nil, err
	}
	host.RTT = time.Since(startTime)

	if resp.GetVersion() != BlobCacheVersion {
		return nil, fmt.Errorf("version mismatch: %s != %s", resp.GetVersion(), BlobCacheVersion)
	}

	return &host, nil
}
