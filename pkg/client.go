package blobcache

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	proto "github.com/beam-cloud/blobcache/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"tailscale.com/client/tailscale"
)

const getContentRequestTimeout = 5 * time.Second
const storeContentRequestTimeout = 60 * time.Second

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
	grpcClient      proto.BlobCacheClient
}

func NewBlobCacheClient(ctx context.Context, cfg BlobCacheConfig) (*BlobCacheClient, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheClientPrefix, uuid.New().String()[:6])
	tailscale := NewTailscale(hostname, cfg)

	server := tailscale.GetOrCreateServer()
	tailscaleClient, err := server.LocalClient()
	if err != nil {
		return nil, err
	}

	bc := &BlobCacheClient{
		ctx:             ctx,
		cfg:             cfg,
		hostname:        hostname,
		tailscale:       tailscale,
		tailscaleClient: tailscaleClient,
	}

	bc.discoveryClient = NewDiscoveryClient(cfg, tailscale, bc.connectToHost)

	// Find and connect to nearest host
	hosts, err := bc.getNearbyHosts()
	if err != nil {
		return nil, err
	}

	go bc.discoveryClient.StartInBackground(bc.ctx)

	err = bc.connectToHost(hosts[0])
	if err != nil {
		return nil, err
	}

	return bc, nil
}

func (c *BlobCacheClient) connectToHost(host *BlobCacheHost) error {
	transportCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())

	token := "" // TODO: add token auth
	isTLS := strings.HasSuffix(host.Addr, "443")
	if isTLS {
		h2creds := credentials.NewTLS(&tls.Config{NextProtos: []string{"h2"}})
		transportCredentials = grpc.WithTransportCredentials(h2creds)
	}

	var dialOpts = []grpc.DialOption{
		transportCredentials,
		grpc.WithContextDialer(c.tailscale.Dial),
	}

	maxMessageSize := c.cfg.GRPCMessageSizeBytes
	if token != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(AuthInterceptor(token)),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(maxMessageSize),
				grpc.MaxCallSendMsgSize(maxMessageSize),
			))
	}

	conn, err := grpc.Dial(host.Addr, dialOpts...)
	if err != nil {
		return err
	}

	c.grpcClient = proto.NewBlobCacheClient(conn)
	return nil
}

func (c *BlobCacheClient) GetContent(hash string, offset int64, length int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(c.ctx, getContentRequestTimeout)
	defer cancel()

	client := c.getGRPCClient()
	getContentResponse, err := client.GetContent(ctx, &proto.GetContentRequest{Hash: hash, Offset: offset, Length: length})
	if err != nil {
		return nil, err
	}
	return getContentResponse.Content, nil
}

func (c *BlobCacheClient) getGRPCClient() proto.BlobCacheClient {
	return c.grpcClient
}

func (c *BlobCacheClient) StoreContent(chunks chan []byte) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, storeContentRequestTimeout)
	defer cancel()

	stream, err := c.grpcClient.StoreContent(ctx)
	if err != nil {
		return "", err
	}

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

	return resp.Hash, nil
}

func (c *BlobCacheClient) GetState() error {
	ctx, cancel := context.WithTimeout(c.ctx, getContentRequestTimeout)
	defer cancel()

	resp, err := c.grpcClient.GetState(ctx, &proto.GetStateRequest{})
	if err != nil {
		return err
	}

	log.Println("resp: ", resp)
	return nil
}

func (c *BlobCacheClient) getNearbyHosts() ([]*BlobCacheHost, error) {
	log.Println("Searching for nearby hosts....")

	maxAttempts := 20
	for attempts := 0; attempts < maxAttempts; attempts++ {
		hosts, err := c.discoveryClient.FindNearbyHosts(context.TODO(), c.tailscaleClient)
		if err != nil {
			return nil, err
		}

		if len(hosts) > 0 {
			log.Println("Located host @ ", hosts[0].Addr)
			return hosts, nil
		}

		time.Sleep(time.Second)
	}

	return nil, errors.New("no host found")
}
