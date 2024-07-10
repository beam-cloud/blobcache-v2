package blobcache

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"tailscale.com/client/tailscale"
)

const getContentRequestTimeout = 30 * time.Second
const storeContentRequestTimeout = 60 * time.Second
const closestHostTimeout = 30 * time.Second
const localClientCacheCleanupInterval = 5 * time.Second
const localClientCacheTTL = 30 * time.Second

func AuthInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		newCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		return invoker(newCtx, method, req, reply, cc, opts...)
	}
}

type BlobCacheClient struct {
	ctx             context.Context
	cfg             BlobCacheConfig
	tailscale       *Tailscale
	hostname        string
	discoveryClient *DiscoveryClient
	tailscaleClient *tailscale.LocalClient
	grpcClients     map[string]proto.BlobCacheClient
	hostMap         *HostMap
	mu              sync.Mutex
	metadata        *BlobCacheMetadata
	closestHost     *BlobCacheHost
	localHostCache  map[string]*localClientCache
}

type localClientCache struct {
	host      *BlobCacheHost
	timestamp time.Time
}

func NewBlobCacheClient(ctx context.Context, cfg BlobCacheConfig) (*BlobCacheClient, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheClientPrefix, uuid.New().String()[:6])
	tailscale := NewTailscale(hostname, cfg)

	InitLogger(cfg.DebugMode)

	server := tailscale.GetOrCreateServer()
	tailscaleClient, err := server.LocalClient()
	if err != nil {
		return nil, err
	}

	metadata, err := NewBlobCacheMetadata(cfg.Metadata)
	if err != nil {
		return nil, err
	}

	bc := &BlobCacheClient{
		ctx:             ctx,
		cfg:             cfg,
		hostname:        hostname,
		tailscale:       tailscale,
		tailscaleClient: tailscaleClient,
		grpcClients:     make(map[string]proto.BlobCacheClient),
		localHostCache:  make(map[string]*localClientCache),
		mu:              sync.Mutex{},
		metadata:        metadata,
		closestHost:     nil,
	}
	bc.hostMap = NewHostMap(bc.addHost)
	bc.discoveryClient = NewDiscoveryClient(cfg, tailscale, bc.hostMap)

	// Start searching for nearby blobcache hosts
	go bc.discoveryClient.StartInBackground(bc.ctx)

	// Monitor and cleanup local client cache
	go bc.manageLocalClientCache(localClientCacheCleanupInterval, localClientCacheTTL)

	return bc, nil
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
					return errors.New("host not found")
				}

				resp, err := client.GetState(c.ctx, &proto.GetStateRequest{})
				if err != nil {
					return errors.New("unable to reach host")
				}

				if resp.GetVersion() != BlobCacheVersion {
					return errors.New("invalid host version")
				}

				return nil
			}()

			// We were unable to reach the host for some reason, remove it as a possible client
			if err != nil {
				c.mu.Lock()
				defer c.mu.Unlock()

				c.hostMap.Remove(host)

				if c.closestHost != nil && c.closestHost.Addr == host.Addr {
					c.closestHost = nil
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

	client, err := c.getGRPCClient(&ClientRequest{
		rt:   ClientRequestTypeRetrieval,
		hash: hash,
	})
	if err != nil {
		return nil, err
	}

	start := time.Now()
	getContentResponse, err := client.GetContent(ctx, &proto.GetContentRequest{Hash: hash, Offset: offset, Length: length})
	if err != nil {
		return nil, err
	}

	Logger.Debugf("Elapsed time to get content: %v", time.Since(start))
	return getContentResponse.Content, nil
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

func (c *BlobCacheClient) getGRPCClient(request *ClientRequest) (proto.BlobCacheClient, error) {
	var host *BlobCacheHost = nil
	var err error = nil

	switch request.rt {
	case ClientRequestTypeStorage:
		if c.closestHost != nil {
			host = c.closestHost
		} else {
			host, err = c.hostMap.Closest(closestHostTimeout)
			if err != nil {
				return nil, err
			}

			c.closestHost = host
		}
	case ClientRequestTypeRetrieval:
		cachedHost, hostFound := c.localHostCache[request.hash]
		if hostFound {
			host = cachedHost.host
		} else {
			hostAddrs, err := c.metadata.GetEntryLocations(c.ctx, request.hash)
			if err != nil {
				return nil, err
			}

			intersection := hostAddrs.Intersect(c.hostMap.Members())
			if intersection.Cardinality() == 0 {
				entry, err := c.metadata.RetrieveEntry(c.ctx, request.hash)
				if err != nil {
					return nil, err
				}

				// Attempt to populate this server with the content from the original source
				if entry.SourcePath != "" {
					Logger.Infof("Content not available in any nearby cache - repopulating from: %s\n", entry.SourcePath)

					host, err = c.hostMap.Closest(closestHostTimeout)
					if err != nil {
						return nil, err
					}

					closestClient, exists := c.grpcClients[host.Addr]
					if !exists {
						return nil, errors.New("client not found")
					}

					resp, err := closestClient.StoreContentFromSource(c.ctx, &proto.StoreContentFromSourceRequest{
						SourcePath:   entry.SourcePath,
						SourceOffset: entry.SourceOffset,
					})
					if err != nil {
						return nil, err
					}

					if resp.Ok {
						return closestClient, nil
					}

					return nil, errors.New("unable to populate content from original source")
				} else {
					return nil, errors.New("no host found")
				}
			}

			host = c.findClosestHost(intersection)
			if host == nil {
				return nil, errors.New("no host found")
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
		return nil, errors.New("no host found")
	}

	client, exists := c.grpcClients[host.Addr]
	if !exists {
		c.mu.Lock()
		defer c.mu.Unlock()
		delete(c.localHostCache, request.hash)
		return nil, errors.New("host not found")

	}

	return client, nil
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

	client, err := c.getGRPCClient(&ClientRequest{
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

	client, err := c.getGRPCClient(&ClientRequest{
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

func (c *BlobCacheClient) GetState() error {
	ctx, cancel := context.WithTimeout(c.ctx, getContentRequestTimeout)
	defer cancel()

	client, err := c.getGRPCClient(&ClientRequest{rt: ClientRequestTypeRetrieval})
	if err != nil {
		return err
	}

	_, err = client.GetState(ctx, &proto.GetStateRequest{})
	if err != nil {
		return err
	}

	return nil
}
