package blobcache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	blobcache_fbs "github.com/beam-cloud/blobcache-v2/proto/blobcache_fbs"
	flatbuffers "github.com/google/flatbuffers/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"tailscale.com/client/tailscale"
)

type DiscoveryClient struct {
	tailscale *Tailscale
	cfg       BlobCacheConfig
	hostMap   *HostMap
	metadata  *BlobCacheMetadata
	mu        sync.Mutex
	tsClient  *tailscale.LocalClient
}

func NewDiscoveryClient(cfg BlobCacheConfig, tailscale *Tailscale, hostMap *HostMap, metadata *BlobCacheMetadata) *DiscoveryClient {
	return &DiscoveryClient{
		cfg:       cfg,
		tailscale: tailscale,
		hostMap:   hostMap,
		metadata:  metadata,
		tsClient:  nil,
	}
}

func (d *DiscoveryClient) updateHostMap(newHosts []*BlobCacheHost) {
	for _, h := range newHosts {
		d.hostMap.Set(h)
	}
}

// Used by blobcache servers to discover their closest peers
func (d *DiscoveryClient) StartInBackground(ctx context.Context) error {
	// Default to metadata discovery if no mode is specified
	if d.cfg.DiscoveryMode == "" {
		d.cfg.DiscoveryMode = string(DiscoveryModeMetadata)
	}

	if d.cfg.DiscoveryMode == string(DiscoveryModeTailscale) {
		server, err := d.tailscale.GetOrCreateServer()
		if err != nil {
			return err
		}

		client, err := server.LocalClient()
		if err != nil {
			return err
		}

		d.tsClient = client
	}

	hosts, err := d.FindNearbyHosts(ctx)
	if err == nil {
		d.updateHostMap(hosts)
	}

	ticker := time.NewTicker(time.Duration(d.cfg.DiscoveryIntervalS) * time.Second)
	for {
		select {
		case <-ticker.C:
			hosts, err := d.FindNearbyHosts(ctx)
			if err != nil {
				continue
			}

			d.updateHostMap(hosts)
		case <-ctx.Done():
			return nil
		}
	}
}

func (d *DiscoveryClient) discoverHostsViaTailscale(ctx context.Context) ([]*BlobCacheHost, error) {
	status, err := d.tsClient.Status(ctx)
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
				Logger.Debugf("Discovered host: %s", hostname)

				defer wg.Done()

				hostname = hostname[:len(hostname)-1] // Strip the last period
				addr := fmt.Sprintf("%s:%d", hostname, d.cfg.Port)

				// Don't try to get the state on peers we're already aware of
				if d.hostMap.Get(addr) != nil {
					return
				}

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

func (d *DiscoveryClient) discoverHostsViaMetadata(ctx context.Context) ([]*BlobCacheHost, error) {
	hosts, err := d.metadata.GetAvailableHosts(ctx, d.hostMap.Remove)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	filteredHosts := []*BlobCacheHost{}
	mu := sync.Mutex{}

	for _, host := range hosts {
		if host.PrivateAddr != "" {
			// Don't try to get the state on peers we're already aware of
			if d.hostMap.Get(host.Addr) != nil {
				continue
			}

			wg.Add(1)
			go func(addr string) {
				defer wg.Done()

				hostState, err := d.GetHostStateViaMetadata(ctx, addr, host.PrivateAddr)
				if err != nil {
					return
				}

				mu.Lock()
				filteredHosts = append(filteredHosts, hostState)
				mu.Unlock()

				Logger.Debugf("Added host with private address to map: %s", hostState.PrivateAddr)
			}(host.Addr)
		}
	}

	wg.Wait()
	return filteredHosts, nil
}

func (d *DiscoveryClient) FindNearbyHosts(ctx context.Context) ([]*BlobCacheHost, error) {
	var hosts []*BlobCacheHost
	var err error

	switch d.cfg.DiscoveryMode {
	case string(DiscoveryModeTailscale):
		hosts, err = d.discoverHostsViaTailscale(ctx)
		if err != nil {
			return nil, err
		}
	case string(DiscoveryModeMetadata):
		hosts, err = d.discoverHostsViaMetadata(ctx)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid discovery mode: %s", d.cfg.DiscoveryMode)
	}

	return hosts, nil
}

// checkService attempts to connect to the gRPC service and verifies its availability
func (d *DiscoveryClient) GetHostState(ctx context.Context, addr string) (*BlobCacheHost, error) {
	host := BlobCacheHost{
		Addr:             addr,
		RTT:              0,
		PrivateAddr:      "",
		CapacityUsagePct: 0.0,
	}

	var dialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(d.tailscale.DialWithTimeout),
		grpc.WithDefaultCallOptions(
			grpc.ForceCodec(flatbuffers.FlatbuffersCodec{}),
		),
	}

	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Query host state to figure out what the round-trip times might look like
	startTime := time.Now()
	c := blobcache_fbs.NewBlobCacheClient(conn)
	b := flatbuffers.NewBuilder(0)
	blobcache_fbs.GetStateRequestStart(b)
	b.Finish(blobcache_fbs.GetStateRequestEnd(b))
	resp, err := c.GetState(ctx, b)
	if err != nil {
		return nil, err
	}

	host.RTT = time.Since(startTime)
	host.CapacityUsagePct = float64(resp.CapacityUsagePct())

	if len(resp.PrivateIpAddr()) > 0 {
		privateAddr := fmt.Sprintf("%s:%d", string(resp.PrivateIpAddr()), d.cfg.Port)
		privateConn, privateErr := DialWithTimeout(ctx, privateAddr)
		if privateErr == nil {
			privateConn.Close()
			host.PrivateAddr = privateAddr
			host.RTT = time.Duration(0)
		}
	}

	threshold := time.Duration(d.cfg.RoundTripThresholdMilliseconds) * time.Millisecond
	if host.RTT > threshold {
		return nil, errors.New("round-trip time exceeds threshold")
	}

	if string(resp.Version()) != BlobCacheVersion {
		return nil, fmt.Errorf("version mismatch: %s != %s", string(resp.Version()), BlobCacheVersion)
	}

	return &host, nil
}

func (d *DiscoveryClient) GetHostStateViaMetadata(ctx context.Context, addr, privateAddr string) (*BlobCacheHost, error) {
	host := BlobCacheHost{
		Addr:             addr,
		RTT:              0,
		PrivateAddr:      privateAddr,
		CapacityUsagePct: 0.0,
	}

	Logger.Infof("Dialing private address: %s", privateAddr)

	var dialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.ForceCodec(flatbuffers.FlatbuffersCodec{}),
		),
	}

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(d.cfg.GRPCDialTimeoutS)*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, privateAddr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := blobcache_fbs.NewBlobCacheClient(conn)

	Logger.Infof("Getting state from %s", privateAddr)

	b := flatbuffers.NewBuilder(0)
	blobcache_fbs.GetStateRequestStart(b)
	b.Finish(blobcache_fbs.GetStateRequestEnd(b))

	resp, err := c.GetState(dialCtx, b)
	if err != nil {
		Logger.Errorf("Error getting state from %s: %v", privateAddr, err)
		return nil, err
	}

	host.RTT = 0
	host.CapacityUsagePct = float64(resp.CapacityUsagePct())

	if string(resp.Version()) != BlobCacheVersion {
		return nil, fmt.Errorf("version mismatch: %s != %s", string(resp.Version()), BlobCacheVersion)
	}

	return &host, nil
}
