package blobcache

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"github.com/google/uuid"
	"github.com/hanwen/go-fuse/v2/fuse"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	getContentRequestTimeout        = 60 * time.Second
	getContentStreamRequestTimeout  = 600 * time.Second
	storeContentRequestTimeout      = 300 * time.Second
	closestHostTimeout              = 30 * time.Second
	localClientCacheCleanupInterval = 5 * time.Second
	localClientCacheTTL             = 300 * time.Second

	// NOTE: This value for readAheadKB is separate from the blobfs config since the FUSE library does
	// weird stuff with the other read_ahead_kb value internally
	readAheadKB = 32768
)

func GrpcAuthInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		newCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}

type BlobCacheClient struct {
	ctx                     context.Context
	clientConfig            BlobCacheClientConfig
	globalConfig            BlobCacheGlobalConfig
	hostname                string
	grpcClients             map[string]proto.BlobCacheClient
	hostMap                 *HostMap
	mu                      sync.RWMutex
	discoveryClient         *DiscoveryClient
	coordinator             CoordinatorClient
	closestHostWithCapacity *BlobCacheHost
	localHostCache          map[string]*localClientCache
	blobfsServer            *fuse.Server
	hasher                  *RendezvousHasher
}

type localClientCache struct {
	host      *BlobCacheHost
	timestamp time.Time
}

func NewBlobCacheClient(ctx context.Context, cfg BlobCacheConfig) (*BlobCacheClient, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheClientPrefix, uuid.New().String()[:6])

	coordinator, err := NewCoordinatorClientRemote(cfg.Global, cfg.Client.Token)
	if err != nil {
		return nil, err
	}

	bc := &BlobCacheClient{
		ctx:                     ctx,
		clientConfig:            cfg.Client,
		globalConfig:            cfg.Global,
		hostname:                hostname,
		grpcClients:             make(map[string]proto.BlobCacheClient),
		localHostCache:          make(map[string]*localClientCache),
		mu:                      sync.RWMutex{},
		coordinator:             coordinator,
		closestHostWithCapacity: nil,
	}

	bc.hostMap = NewHostMap(cfg.Global, bc.addHost)
	bc.discoveryClient = NewDiscoveryClient(cfg.Global, bc.hostMap, coordinator)
	bc.hasher = NewRendezvousHasher()

	// Start searching for nearby blobcache hosts
	go bc.discoveryClient.Start(bc.ctx)

	// Monitor and cleanup local client cache
	go bc.manageLocalClientCache(localClientCacheCleanupInterval, localClientCacheTTL)

	// Mount cache as a FUSE filesystem if blobfs is enabled
	if bc.clientConfig.BlobFs.Enabled {
		startServer, _, server, err := Mount(ctx, BlobFsSystemOpts{
			Config:            cfg,
			CoordinatorClient: coordinator,
			Client:            bc,
			Verbose:           bc.globalConfig.DebugMode,
		})
		if err != nil {
			return nil, err
		}

		err = startServer()
		if err != nil {
			return nil, err
		}

		err = updateReadAheadKB(bc.clientConfig.BlobFs.MountPoint, readAheadKB)
		if err != nil {
			Logger.Errorf("Failed to update read_ahead_kb: %v", err)
		}

		bc.blobfsServer = server
	}

	return bc, nil
}

func (c *BlobCacheClient) Cleanup() error {
	if c.clientConfig.BlobFs.Enabled && c.blobfsServer != nil {
		c.blobfsServer.Unmount()
	}
	return nil
}

func (c *BlobCacheClient) GetNearbyHosts() ([]*BlobCacheHost, error) {
	hosts, err := c.coordinator.GetAvailableHosts(c.ctx, "myregion")
	if err != nil {
		return nil, err
	}

	return hosts, nil
}

func (c *BlobCacheClient) addHost(host *BlobCacheHost) error {
	transportCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())

	isTLS := c.globalConfig.TLSEnabled
	if isTLS {
		h2creds := credentials.NewTLS(&tls.Config{NextProtos: []string{"h2"}})
		transportCredentials = grpc.WithTransportCredentials(h2creds)
	}

	var dialFunc func(context.Context, string) (net.Conn, error) = nil
	addr := host.Addr

	dialFunc = DialWithTimeout
	if host.PrivateAddr != "" {
		addr = host.PrivateAddr
	}

	maxMessageSize := c.globalConfig.GRPCMessageSizeBytes
	var dialOpts = []grpc.DialOption{
		transportCredentials,
		grpc.WithContextDialer(dialFunc),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
			grpc.MaxCallSendMsgSize(maxMessageSize),
		),
	}

	if c.clientConfig.Token != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(GrpcAuthInterceptor(c.clientConfig.Token)))
	}

	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.grpcClients[host.Host] = proto.NewBlobCacheClient(conn)
	c.hasher.Add(host)

	go c.monitorHost(host)
	return nil
}

func (c *BlobCacheClient) monitorHost(host *BlobCacheHost) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := func() error {
				client, exists := c.grpcClients[host.Host]
				if !exists {
					return ErrHostNotFound
				}

				resp, err := client.GetState(c.ctx, &proto.GetStateRequest{})
				if err != nil {
					return ErrInvalidHostVersion
				}

				if resp.GetVersion() != BlobCacheVersion {
					return ErrInvalidHostVersion
				}

				return nil
			}()

			// We were unable to reach the host for some reason, remove it as a possible client
			if err != nil {
				c.mu.Lock()
				defer c.mu.Unlock()

				c.hostMap.Remove(host)
				c.hasher.Remove(host)

				if c.closestHostWithCapacity != nil && c.closestHostWithCapacity.Addr == host.Addr {
					c.closestHostWithCapacity = nil
				}

				delete(c.grpcClients, host.Addr)

				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

func (c *BlobCacheClient) GetContent(hash string, offset int64, length int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(c.ctx, getContentRequestTimeout)
	defer cancel()

	for attempt := 0; attempt < 3; attempt += 1 {
		client, _, err := c.getGRPCClient(ctx, &ClientRequest{
			rt:   ClientRequestTypeRetrieval,
			hash: hash,
		})
		if err != nil {
			return nil, err
		}

		start := time.Now()
		getContentResponse, err := client.GetContent(ctx, &proto.GetContentRequest{Hash: hash, Offset: offset, Length: length})
		if err != nil || !getContentResponse.Ok {
			c.mu.Lock()
			delete(c.localHostCache, hash)
			c.mu.Unlock()

			continue
		}

		Logger.Debugf("Elapsed time to get content: %v", time.Since(start))
		return getContentResponse.Content, nil
	}

	return nil, ErrContentNotFound
}

func (c *BlobCacheClient) GetContentStream(hash string, offset int64, length int64) (chan []byte, error) {
	ctx, cancel := context.WithTimeout(c.ctx, getContentStreamRequestTimeout)
	contentChan := make(chan []byte)

	go func() {
		defer close(contentChan)
		defer cancel()

		for attempt := 0; attempt < 3; attempt++ {
			client, _, err := c.getGRPCClient(ctx, &ClientRequest{
				rt:   ClientRequestTypeRetrieval,
				hash: hash,
			})
			if err != nil {
				return
			}

			stream, err := client.GetContentStream(ctx, &proto.GetContentRequest{Hash: hash, Offset: offset, Length: length})
			if err != nil {
				c.mu.Lock()
				delete(c.localHostCache, hash)
				c.mu.Unlock()
				continue
			}

			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					return
				}

				if err != nil || !resp.Ok {
					c.mu.Lock()
					delete(c.localHostCache, hash)
					c.mu.Unlock()
					break
				}

				contentChan <- resp.Content
			}
		}
	}()

	return contentChan, nil
}

func (c *BlobCacheClient) manageLocalClientCache(ttl time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				now := time.Now()
				stale := make([]string, 0)

				c.mu.RLock()
				for hash, entry := range c.localHostCache {
					if now.Sub(entry.timestamp) > ttl {
						stale = append(stale, hash)
					}
				}
				c.mu.RUnlock()
				c.mu.Lock()
				for _, hash := range stale {
					delete(c.localHostCache, hash)
				}
				c.mu.Unlock()

			case <-c.ctx.Done():
				return
			}
		}
	}()
}

func (c *BlobCacheClient) IsCachedNearby(ctx context.Context, hash string) bool {
	return false
}

func (c *BlobCacheClient) getGRPCClient(ctx context.Context, request *ClientRequest) (proto.BlobCacheClient, *BlobCacheHost, error) {
	var host *BlobCacheHost = nil

	hostIndex := 0
	switch request.rt {
	case ClientRequestTypeStorage:
		// TODO: Make N configurable and cycle through hosts based on attempt number
		hosts := c.hasher.GetN(3, request.hash)

		if len(hosts) > 0 {
			host = hosts[hostIndex]
		}

	case ClientRequestTypeRetrieval:
		c.mu.RLock()
		cachedHost, hostFound := c.localHostCache[request.hash]
		c.mu.RUnlock()

		if hostFound {
			host = cachedHost.host
		} else {
			c.mu.Lock()

			// TODO: Make N configurable and cycle through hosts based on attempt number
			hosts := c.hasher.GetN(3, request.hash)

			if len(hosts) == 0 {
				host = nil
			} else {
				host = hosts[hostIndex]
				c.localHostCache[request.hash] = &localClientCache{
					host:      host,
					timestamp: time.Now(),
				}
			}

			c.mu.Unlock()

		}
	default:
	}

	if host == nil {
		return nil, nil, ErrHostNotFound
	}

	client, exists := c.grpcClients[host.Host]
	if !exists {
		c.mu.Lock()
		delete(c.localHostCache, request.hash)
		c.mu.Unlock()
		return nil, nil, ErrHostNotFound
	}

	return client, host, nil
}

func (c *BlobCacheClient) StoreContent(chunks chan []byte, hash string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, storeContentRequestTimeout)
	defer cancel()

	client, _, err := c.getGRPCClient(ctx, &ClientRequest{
		rt:   ClientRequestTypeStorage,
		hash: hash,
	})
	if err != nil {
		return "", err
	}

	stream, err := client.StoreContent(ctx)
	if err != nil {
		return "", err
	}

	start := time.Now()
	for chunk := range chunks {
		req := &proto.StoreContentRequest{Content: chunk, Hash: hash}
		if err := stream.Send(req); err != nil {
			return "", err
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", err
	}

	Logger.Debugf("Elapsed time to send content: %v", time.Since(start))
	return resp.Hash, nil
}

func (c *BlobCacheClient) StoreContentFromSource(sourcePath string, sourceOffset int64, hash string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, storeContentRequestTimeout)
	defer cancel()

	client, _, err := c.getGRPCClient(ctx, &ClientRequest{
		rt: ClientRequestTypeStorage,
	})
	if err != nil {
		return "", err
	}

	resp, err := client.StoreContentFromSource(ctx, &proto.StoreContentFromSourceRequest{SourcePath: sourcePath, SourceOffset: sourceOffset})
	if err != nil {
		return "", err
	}

	return resp.Hash, nil
}

func (c *BlobCacheClient) StoreContentFromSourceWithLock(sourcePath string, sourceOffset int64) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, storeContentRequestTimeout)
	defer cancel()

	client, _, err := c.getGRPCClient(ctx, &ClientRequest{
		rt: ClientRequestTypeStorage,
	})
	if err != nil {
		return "", err
	}

	resp, err := client.StoreContentFromSourceWithLock(ctx, &proto.StoreContentFromSourceRequest{SourcePath: sourcePath, SourceOffset: sourceOffset})
	if err != nil {
		return "", err
	}

	if resp.FailedToAcquireLock {
		return "", ErrUnableToAcquireLock
	}

	return resp.Hash, nil

}

func (c *BlobCacheClient) HostsAvailable() bool {
	return c.hostMap.Members().Cardinality() > 0
}

func (c *BlobCacheClient) WaitForHosts(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if c.HostsAvailable() {
		Logger.Infof("Cache available.")
		return nil
	}

	Logger.Infof("Waiting for hosts to be available...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if c.HostsAvailable() {
				Logger.Infof("Cache available.")
				return nil
			}

			time.Sleep(1 * time.Second)
		}
	}
}

func (c *BlobCacheClient) GetState() error {
	ctx, cancel := context.WithTimeout(c.ctx, getContentRequestTimeout)
	defer cancel()

	client, _, err := c.getGRPCClient(ctx, &ClientRequest{rt: ClientRequestTypeRetrieval})
	if err != nil {
		return err
	}

	_, err = client.GetState(ctx, &proto.GetStateRequest{})
	if err != nil {
		return err
	}

	return nil
}
