package blobcache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DiscoveryClient struct {
	cfg         BlobCacheGlobalConfig
	hostMap     *HostMap
	coordinator *CoordinatorClient
	mu          sync.Mutex
}

func NewDiscoveryClient(cfg BlobCacheGlobalConfig, hostMap *HostMap, coordinator *CoordinatorClient) *DiscoveryClient {
	return &DiscoveryClient{
		cfg:         cfg,
		hostMap:     hostMap,
		coordinator: coordinator,
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

func (d *DiscoveryClient) discoverHostsViaMetadata(ctx context.Context) ([]*BlobCacheHost, error) {
	hosts, err := d.coordinator.GetAvailableHosts(ctx, "testmeout")
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
		grpc.WithContextDialer(DialWithTimeout),
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
	host.CapacityUsagePct = float64(resp.GetCapacityUsagePct())

	if resp.PrivateIpAddr != "" {
		privateAddr := fmt.Sprintf("%s:%d", resp.PrivateIpAddr, d.cfg.ServerPort)
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

	if resp.GetVersion() != BlobCacheVersion {
		return nil, fmt.Errorf("version mismatch: %s != %s", resp.GetVersion(), BlobCacheVersion)
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

	var dialOpts = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(d.cfg.GRPCDialTimeoutS)*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, privateAddr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := proto.NewBlobCacheClient(conn)

	resp, err := c.GetState(dialCtx, &proto.GetStateRequest{})
	if err != nil {
		return nil, err
	}

	host.RTT = 0
	host.CapacityUsagePct = float64(resp.GetCapacityUsagePct())

	if resp.GetVersion() != BlobCacheVersion {
		return nil, fmt.Errorf("version mismatch: %s != %s", resp.GetVersion(), BlobCacheVersion)
	}

	return &host, nil
}
