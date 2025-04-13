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

func (c *CoordinatorClient) StoreContentInBlobFs(ctx context.Context, path string, hash string, size uint64) error {
	return nil
}

func (c *CoordinatorClient) GetContentFromBlobFs(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (c *CoordinatorClient) ListDirectory(ctx context.Context, path string) ([]string, error) {
	return nil, nil
}
