package blobcache

import (
	"context"
	"crypto/tls"
	"io"
	"sync"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	rendezvous "github.com/beam-cloud/rendezvous"
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

type RendezvousHasher interface {
	Add(hosts ...*BlobCacheHost)
	Remove(host *BlobCacheHost)
	GetN(n int, key string) []*BlobCacheHost
}

type BlobCacheClient struct {
	ctx                   context.Context
	locality              string
	clientConfig          BlobCacheClientConfig
	globalConfig          BlobCacheGlobalConfig
	grpcClients           map[string]proto.BlobCacheClient
	hostMap               *HostMap
	mu                    sync.RWMutex
	discoveryClient       *DiscoveryClient
	coordinator           CoordinatorClient
	localHostCache        map[string]*localClientCache
	blobfsServer          *fuse.Server
	hasher                RendezvousHasher
	minRetryLengthBytes   int64
	maxGetContentAttempts int64
}

type localClientCache struct {
	host      *BlobCacheHost
	timestamp time.Time
}

func NewBlobCacheClient(ctx context.Context, cfg BlobCacheConfig) (*BlobCacheClient, error) {
	InitLogger(cfg.Global.DebugMode, cfg.Global.PrettyLogs)

	coordinator, err := NewCoordinatorClientRemote(cfg.Global, cfg.Client.Token)
	if err != nil {
		return nil, err
	}

	locality := cfg.Global.GetLocality()

	bc := &BlobCacheClient{
		ctx:                   ctx,
		locality:              locality,
		clientConfig:          cfg.Client,
		globalConfig:          cfg.Global,
		grpcClients:           make(map[string]proto.BlobCacheClient),
		localHostCache:        make(map[string]*localClientCache),
		mu:                    sync.RWMutex{},
		coordinator:           coordinator,
		hasher:                rendezvous.New[*BlobCacheHost](),
		minRetryLengthBytes:   cfg.Client.MinRetryLengthBytes,
		maxGetContentAttempts: max(int64(cfg.Client.MaxGetContentAttempts), 1),
	}

	bc.hostMap = NewHostMap(cfg.Global, bc.addHost)
	bc.discoveryClient = NewDiscoveryClient(cfg.Global, bc.hostMap, coordinator, locality)

	// Start searching for nearby blobcache hosts
	go bc.discoveryClient.Start(bc.ctx)

	// Monitor and cleanup local client cache
	go bc.manageLocalClientCache(localClientCacheCleanupInterval, localClientCacheTTL)

	// Mount cache as a FUSE filesystem if blobfs is enabled
	if bc.clientConfig.BlobFs.Enabled {
		startServer, _, server, err := Mount(ctx, BlobFsSystemOpts{
			Config:            cfg.Client,
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
	hosts, err := c.coordinator.GetAvailableHosts(c.ctx, c.locality)
	if err != nil {
		return nil, err
	}

	return hosts, nil
}

func (c *BlobCacheClient) addHost(host *BlobCacheHost) error {
	addr := host.Addr
	if host.PrivateAddr != "" {
		addr = host.PrivateAddr
	}

	transportCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())
	if isTLSEnabled(addr) {
		h2creds := credentials.NewTLS(&tls.Config{NextProtos: []string{"h2"}})
		transportCredentials = grpc.WithTransportCredentials(h2creds)
	}

	var dialOpts = []grpc.DialOption{
		transportCredentials,
		grpc.WithContextDialer(DialWithTimeout),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(c.globalConfig.GRPCMessageSizeBytes),
			grpc.MaxCallSendMsgSize(c.globalConfig.GRPCMessageSizeBytes),
		),
	}

	if c.clientConfig.Token != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(grpcAuthInterceptor(c.clientConfig.Token)))
	}

	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.grpcClients[host.HostId] = proto.NewBlobCacheClient(conn)
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
				client, exists := c.grpcClients[host.HostId]
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

				delete(c.grpcClients, host.Addr)

				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

func (c *BlobCacheClient) IsPathCachedNearby(ctx context.Context, path string) bool {
	metadata, err := c.coordinator.GetFsNode(ctx, GenerateFsID(path))
	if err != nil {
		Logger.Errorf("error getting fs node: %v, path: %s", err, path)
		return false
	}

	exists, err := c.IsCachedNearby(metadata.Hash)
	if err != nil {
		Logger.Errorf("error checking if content is cached nearby: %v, hash: %s", err, metadata.Hash)
		return false
	}

	return exists
}

func (c *BlobCacheClient) IsCachedNearby(hash string) (bool, error) {
	hostsToCheck := c.clientConfig.NTopHosts

	for hostIndex := 0; hostIndex < hostsToCheck; hostIndex++ {
		client, _, err := c.getGRPCClient(&ClientRequest{
			rt:        ClientRequestTypeRetrieval,
			hash:      hash,
			key:       hash,
			hostIndex: hostIndex,
		})
		if err != nil {
			return false, err
		}

		resp, err := client.HasContent(c.ctx, &proto.HasContentRequest{Hash: hash})
		if err != nil {
			return false, err
		}

		if resp.Exists {
			return true, nil
		}
	}

	return false, nil
}

func (c *BlobCacheClient) GetContent(hash string, offset int64, length int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(c.ctx, getContentRequestTimeout)
	defer cancel()

	maxAttempts := 1
	if length > c.minRetryLengthBytes {
		maxAttempts = int(c.maxGetContentAttempts)
	}

	for attempt := 0; attempt < maxAttempts; attempt += 1 {
		client, _, err := c.getGRPCClient(&ClientRequest{
			rt:        ClientRequestTypeRetrieval,
			hash:      hash,
			key:       hash,
			hostIndex: attempt,
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

		maxAttempts := 1
		if length > c.minRetryLengthBytes {
			maxAttempts = int(c.maxGetContentAttempts)
		}

		for attempt := 0; attempt < maxAttempts; attempt++ {
			client, _, err := c.getGRPCClient(&ClientRequest{
				rt:        ClientRequestTypeRetrieval,
				hash:      hash,
				key:       hash,
				hostIndex: attempt,
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

func (c *BlobCacheClient) getGRPCClient(request *ClientRequest) (proto.BlobCacheClient, *BlobCacheHost, error) {
	var host *BlobCacheHost = nil

	switch request.rt {
	case ClientRequestTypeStorage:
		hosts := c.hasher.GetN(c.clientConfig.NTopHosts, request.key)
		hostIndex := min(int64(request.hostIndex), int64(len(hosts)-1))

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

			hosts := c.hasher.GetN(c.clientConfig.NTopHosts, request.key)
			hostIndex := min(int64(request.hostIndex), int64(len(hosts)-1))

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

	client, exists := c.grpcClients[host.HostId]
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

	client, _, err := c.getGRPCClient(&ClientRequest{
		rt:        ClientRequestTypeStorage,
		hash:      hash,
		key:       hash,
		hostIndex: 0,
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
		req := &proto.StoreContentRequest{Content: chunk}
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

func (c *BlobCacheClient) StoreContentFromSource(sourcePath string, sourceOffset int64, bucketName string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, storeContentRequestTimeout)
	defer cancel()

	client, _, err := c.getGRPCClient(&ClientRequest{
		rt:        ClientRequestTypeStorage,
		key:       sourcePath,
		hostIndex: 0,
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

func (c *BlobCacheClient) StoreContentFromSourceWithLock(sourcePath string, sourceOffset int64, bucketName string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, storeContentRequestTimeout)
	defer cancel()

	client, _, err := c.getGRPCClient(&ClientRequest{
		rt:        ClientRequestTypeStorage,
		key:       sourcePath,
		hostIndex: 0,
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

	if !resp.Ok {
		return "", ErrUnableToPopulateContent
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

	client, _, err := c.getGRPCClient(&ClientRequest{rt: ClientRequestTypeRetrieval, hostIndex: 0})
	if err != nil {
		return err
	}

	_, err = client.GetState(ctx, &proto.GetStateRequest{})
	if err != nil {
		return err
	}

	return nil
}

func grpcAuthInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		newCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}
