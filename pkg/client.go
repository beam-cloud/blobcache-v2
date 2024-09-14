package blobcache

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	"github.com/hanwen/go-fuse/v2/fuse"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"tailscale.com/client/tailscale"
)

const getContentRequestTimeout = 30 * time.Second
const storeContentRequestTimeout = 300 * time.Second
const closestHostTimeout = 30 * time.Second
const localClientCacheCleanupInterval = 5 * time.Second
const localClientCacheTTL = 300 * time.Second

func AuthInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		newCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}

type BlobCacheClient struct {
	ctx                     context.Context
	cfg                     BlobCacheConfig
	tailscale               *Tailscale
	hostname                string
	discoveryClient         *DiscoveryClient
	tailscaleClient         *tailscale.LocalClient
	grpcClients             map[string]proto.BlobCacheClient
	hostMap                 *HostMap
	mu                      sync.Mutex
	metadata                *BlobCacheMetadata
	closestHostWithCapacity *BlobCacheHost
	localHostCache          map[string]*localClientCache
	blobfsServer            *fuse.Server
}

type localClientCache struct {
	host      *BlobCacheHost
	timestamp time.Time
}

func NewBlobCacheClient(ctx context.Context, cfg BlobCacheConfig) (*BlobCacheClient, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheClientPrefix, uuid.New().String()[:6])
	tailscale := NewTailscale(ctx, hostname, cfg)

	InitLogger(cfg.DebugMode)

	server, err := tailscale.GetOrCreateServer()
	if err != nil {
		return nil, err
	}

	tailscaleClient, err := server.LocalClient()
	if err != nil {
		return nil, err
	}

	// Block until tailscale is authenticated (or the timeout is reached)
	err = tailscale.WaitForAuth(ctx, time.Second*30)
	if err != nil {
		return nil, err
	}

	metadata, err := NewBlobCacheMetadata(cfg.Metadata)
	if err != nil {
		return nil, err
	}

	bc := &BlobCacheClient{
		ctx:                     ctx,
		cfg:                     cfg,
		hostname:                hostname,
		tailscale:               tailscale,
		tailscaleClient:         tailscaleClient,
		grpcClients:             make(map[string]proto.BlobCacheClient),
		localHostCache:          make(map[string]*localClientCache),
		mu:                      sync.Mutex{},
		metadata:                metadata,
		closestHostWithCapacity: nil,
	}
	bc.hostMap = NewHostMap(cfg, bc.addHost)
	bc.discoveryClient = NewDiscoveryClient(cfg, tailscale, bc.hostMap)

	// Start searching for nearby blobcache hosts
	go bc.discoveryClient.StartInBackground(bc.ctx)

	// Monitor and cleanup local client cache
	go bc.manageLocalClientCache(localClientCacheCleanupInterval, localClientCacheTTL)

	// Mount cache as a FUSE filesystem if blobfs is enabled
	if cfg.BlobFs.Enabled {
		startServer, _, server, err := Mount(ctx, BlobFsSystemOpts{
			Config:   cfg,
			Metadata: metadata,
			Client:   bc,
			Verbose:  cfg.DebugMode,
		})
		if err != nil {
			return nil, err
		}

		err = startServer()
		if err != nil {
			return nil, err
		}

		bc.blobfsServer = server
	}

	return bc, nil
}

func (c *BlobCacheClient) Cleanup() error {
	if c.cfg.BlobFs.Enabled && c.blobfsServer != nil {
		c.blobfsServer.Unmount()
	}
	return nil
}

func (c *BlobCacheClient) addHost(host *BlobCacheHost) error {
	transportCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())

	isTLS := c.cfg.TLSEnabled
	if isTLS {
		h2creds := credentials.NewTLS(&tls.Config{NextProtos: []string{"h2"}})
		transportCredentials = grpc.WithTransportCredentials(h2creds)
	}

	var dialFunc func(context.Context, string) (net.Conn, error) = nil
	addr := host.Addr
	dialFunc = c.tailscale.DialWithTimeout
	if host.PrivateAddr != "" {
		dialFunc = DialWithTimeout
		addr = host.PrivateAddr
	}

	maxMessageSize := c.cfg.GRPCMessageSizeBytes
	var dialOpts = []grpc.DialOption{
		transportCredentials,
		grpc.WithContextDialer(dialFunc),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
			grpc.MaxCallSendMsgSize(maxMessageSize),
		),
	}

	if c.cfg.Token != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(AuthInterceptor(c.cfg.Token)))
	}

	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.grpcClients[host.Addr] = proto.NewBlobCacheClient(conn)

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
				client, exists := c.grpcClients[host.Addr]
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
		client, host, err := c.getGRPCClient(ctx, &ClientRequest{
			rt:   ClientRequestTypeRetrieval,
			hash: hash,
		})
		if err != nil {
			return nil, err
		}

		start := time.Now()
		getContentResponse, err := client.GetContent(ctx, &proto.GetContentRequest{Hash: hash, Offset: offset, Length: length})
		if err != nil {
			// If we had an issue getting the content, remove this location from metadata
			c.metadata.RemoveEntryLocation(ctx, hash, host)

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

func (c *BlobCacheClient) manageLocalClientCache(ttl time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				now := time.Now()

				for hash, entry := range c.localHostCache {
					if now.Sub(entry.timestamp) > ttl {
						c.mu.Lock()
						delete(c.localHostCache, hash)
						c.mu.Unlock()
					}
				}

			case <-c.ctx.Done():
				return
			}
		}
	}()
}

func (c *BlobCacheClient) getGRPCClient(ctx context.Context, request *ClientRequest) (proto.BlobCacheClient, *BlobCacheHost, error) {
	var host *BlobCacheHost = nil
	var err error = nil

	switch request.rt {
	case ClientRequestTypeStorage:
		if c.closestHostWithCapacity != nil {
			host = c.closestHostWithCapacity
		} else {
			host, err = c.hostMap.ClosestWithCapacity(closestHostTimeout)
			if err != nil {
				return nil, nil, err
			}

			c.closestHostWithCapacity = host
		}
	case ClientRequestTypeRetrieval:
		cachedHost, hostFound := c.localHostCache[request.hash]
		if hostFound {
			host = cachedHost.host
		} else {
			hostAddrs, err := c.metadata.GetEntryLocations(ctx, request.hash)
			if err != nil {
				return nil, nil, err
			}

			intersection := hostAddrs.Intersect(c.hostMap.Members())
			if intersection.Cardinality() == 0 {
				entry, err := c.metadata.RetrieveEntry(ctx, request.hash)
				if err != nil {
					return nil, nil, err
				}

				// Attempt to populate this server with the content from the original source
				if entry.SourcePath != "" {
					err := c.metadata.SetClientLock(ctx, c.hostname, request.hash)
					if err != nil {
						return nil, nil, ErrCacheLockHeld
					}
					defer c.metadata.RemoveClientLock(ctx, c.hostname, request.hash)

					Logger.Infof("Content not available in any nearby cache - repopulating from: %s\n", entry.SourcePath)
					host, err = c.hostMap.Closest(closestHostTimeout)
					if err != nil {
						return nil, nil, err
					}

					closestClient, exists := c.grpcClients[host.Addr]
					if !exists {
						return nil, nil, ErrClientNotFound
					}

					resp, err := closestClient.StoreContentFromSource(c.ctx, &proto.StoreContentFromSourceRequest{
						SourcePath:   entry.SourcePath,
						SourceOffset: entry.SourceOffset,
					})
					if err != nil {
						return nil, nil, err
					}

					if resp.Ok {
						return closestClient, host, nil
					}

					return nil, nil, ErrUnableToPopulateContent
				} else {
					return nil, nil, ErrHostNotFound
				}
			}

			host = c.findClosestHost(intersection)
			if host == nil {
				return nil, nil, ErrHostNotFound
			}

			c.mu.Lock()
			c.localHostCache[request.hash] = &localClientCache{
				host:      host,
				timestamp: time.Now(),
			}
			c.mu.Unlock()

		}
	default:
	}

	if host == nil {
		return nil, nil, ErrHostNotFound
	}

	client, exists := c.grpcClients[host.Addr]
	if !exists {
		c.mu.Lock()
		delete(c.localHostCache, request.hash)
		c.mu.Unlock()
		return nil, nil, ErrHostNotFound
	}

	return client, host, nil
}

func (c *BlobCacheClient) findClosestHost(intersection mapset.Set[string]) *BlobCacheHost {
	var closestHost *BlobCacheHost
	closestRTT := time.Duration(math.MaxInt64)

	for addr := range intersection.Iter() {
		host := c.hostMap.Get(addr)

		if host != nil && host.RTT < closestRTT {
			closestHost = host
			closestRTT = host.RTT
		}
	}

	return closestHost
}

func (c *BlobCacheClient) StoreContent(chunks chan []byte) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, storeContentRequestTimeout)
	defer cancel()

	client, _, err := c.getGRPCClient(ctx, &ClientRequest{
		rt: ClientRequestTypeStorage,
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

	Logger.Debugf("Elapsed time to send content: %v\n", time.Since(start))
	return resp.Hash, nil
}

func (c *BlobCacheClient) StoreContentFromSource(sourcePath string, sourceOffset int64) (string, error) {
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

func (c *BlobCacheClient) HostsAvailable() bool {
	return c.hostMap.Members().Cardinality() > 0
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
