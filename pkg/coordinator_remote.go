package blobcache

import (
	"context"
	"crypto/tls"
	"net"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type CoordinatorClientRemote struct {
	host   string
	cfg    BlobCacheGlobalConfig
	client proto.BlobCacheClient
}

func NewCoordinatorClientRemote(cfg BlobCacheGlobalConfig, token string) (CoordinatorClient, error) {
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

	return &CoordinatorClientRemote{cfg: cfg, host: cfg.CoordinatorHost, client: proto.NewBlobCacheClient(conn)}, nil
}

func (c *CoordinatorClientRemote) AddHostToIndex(ctx context.Context, locality string, host *BlobCacheHost) error {
	return nil
}

func (c *CoordinatorClientRemote) SetHostKeepAlive(ctx context.Context, locality string, host *BlobCacheHost) error {
	return nil
}

func (c *CoordinatorClientRemote) SetStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *CoordinatorClientRemote) RemoveStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *CoordinatorClientRemote) RefreshStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *CoordinatorClientRemote) GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	response, err := c.client.GetAvailableHosts(ctx, &proto.GetAvailableHostsRequest{Locality: locality})
	if err != nil {
		return nil, err
	}

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

func (c *CoordinatorClientRemote) SetClientLock(ctx context.Context, hash string, host string) error {
	_, err := c.client.SetClientLock(ctx, &proto.SetClientLockRequest{Hash: hash, Host: host})
	return err
}

func (c *CoordinatorClientRemote) RemoveClientLock(ctx context.Context, hash string, host string) error {
	_, err := c.client.RemoveClientLock(ctx, &proto.RemoveClientLockRequest{Hash: hash, Host: host})
	return err
}

func (c *CoordinatorClientRemote) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	// _, err := c.client.SetFsNode(ctx, &proto.SetFsNodeRequest{Id: id, Path: metadata.Path, Hash: metadata.Hash, Size: metadata.Size})
	return nil
}

func (c *CoordinatorClientRemote) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
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

func (c *CoordinatorClientRemote) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	return nil, nil
}
