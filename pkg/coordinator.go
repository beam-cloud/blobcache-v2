package blobcache

import (
	"context"
	"crypto/tls"
	"net"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	mapset "github.com/deckarep/golang-set/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type CoordinatorClient struct {
	cfg    BlobCacheGlobalConfig
	client proto.BlobCacheClient
	host   string
}

func NewCoordinatorClient(cfg BlobCacheGlobalConfig, token string) (*CoordinatorClient, error) {
	transportCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())

	isTLS := cfg.TLSEnabled
	if isTLS {
		h2creds := credentials.NewTLS(&tls.Config{NextProtos: []string{"h2"}})
		transportCredentials = grpc.WithTransportCredentials(h2creds)
	}

	var dialFunc func(context.Context, string) (net.Conn, error) = nil
	addr := cfg.CoordinatorHost

	dialFunc = DialWithTimeout

	maxMessageSize := cfg.GRPCMessageSizeBytes
	var dialOpts = []grpc.DialOption{
		transportCredentials,
		grpc.WithContextDialer(dialFunc),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
			grpc.MaxCallSendMsgSize(maxMessageSize),
		),
	}

	if token != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(GrpcAuthInterceptor(token)))
	}

	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}

	return &CoordinatorClient{cfg: cfg, host: cfg.CoordinatorHost, client: proto.NewBlobCacheClient(conn)}, nil
}

func (c *CoordinatorClient) GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	response, err := c.client.GetAvailableHosts(ctx, &proto.GetAvailableHostsRequest{Locality: locality})
	if err != nil {
		return nil, err
	}

	Logger.Infof("Hosts: %v", response.Hosts)

	hosts := make([]*BlobCacheHost, 0)
	for _, host := range response.Hosts {
		hosts = append(hosts, &BlobCacheHost{
			Host:        host.Host,
			Addr:        host.Addr,
			PrivateAddr: host.PrivateIpAddr,
		})
	}

	return hosts, nil
}

func (c *CoordinatorClient) GetEntryLocations(ctx context.Context, hash string) (mapset.Set[string], error) {
	hostAddrs := []string{}

	hostSet := mapset.NewSet[string]()
	for _, addr := range hostAddrs {
		hostSet.Add(addr)
	}

	return hostSet, nil
}

func (c *CoordinatorClient) SetClientLock(ctx context.Context, hash string, host string) error {
	_, err := c.client.SetClientLock(ctx, &proto.SetClientLockRequest{Hash: hash, Host: host})
	return err
}

func (c *CoordinatorClient) RemoveClientLock(ctx context.Context, hash string, host string) error {
	_, err := c.client.RemoveClientLock(ctx, &proto.RemoveClientLockRequest{Hash: hash, Host: host})
	return err
}

func (c *CoordinatorClient) RetrieveEntry(ctx context.Context, hash string) (*BlobCacheEntry, error) {
	return nil, &ErrEntryNotFound{Hash: hash}
}

func (c *CoordinatorClient) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	// _, err := c.client.SetFsNode(ctx, &proto.SetFsNodeRequest{Id: id, Path: metadata.Path, Hash: metadata.Hash, Size: metadata.Size})
	return nil
}

func (c *CoordinatorClient) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	response, err := c.client.GetFsNode(ctx, &proto.GetFsNodeRequest{Id: id})
	if err != nil {
		return nil, err
	}

	return &BlobFsMetadata{
		Path: response.Path,
		Hash: response.Hash,
		Size: response.Size,
	}, nil
}

func (c *CoordinatorClient) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	return nil, nil
}
