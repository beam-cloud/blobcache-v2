package blobcache

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
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
	grpcClients     map[string]proto.BlobCacheClient
	hostMap         *HostMap
	mu              sync.Mutex
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
		grpcClients:     make(map[string]proto.BlobCacheClient),
		mu:              sync.Mutex{},
	}
	bc.hostMap = NewHostMap(bc.connectToHost)
	bc.discoveryClient = NewDiscoveryClient(cfg, tailscale, bc.hostMap)

	// Start searching for nearby hosts
	go bc.discoveryClient.StartInBackground(bc.ctx)
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

	maxMessageSize := c.cfg.GRPCMessageSizeBytes
	var dialOpts = []grpc.DialOption{
		transportCredentials,
		grpc.WithContextDialer(c.tailscale.Dial),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageSize),
			grpc.MaxCallSendMsgSize(maxMessageSize),
		),
	}

	if token != "" {
		dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(AuthInterceptor(token)))
	}

	conn, err := grpc.Dial(host.Addr, dialOpts...)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.grpcClients[host.Addr] = proto.NewBlobCacheClient(conn)
	return nil
}

func (c *BlobCacheClient) GetContent(hash string, offset int64, length int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(c.ctx, getContentRequestTimeout)
	defer cancel()

	client, err := c.getGRPCClient()
	if err != nil {
		return nil, err
	}

	getContentResponse, err := client.GetContent(ctx, &proto.GetContentRequest{Hash: hash, Offset: offset, Length: length})
	if err != nil {
		return nil, err
	}
	return getContentResponse.Content, nil
}

func (c *BlobCacheClient) getGRPCClient() (proto.BlobCacheClient, error) {
	host, err := c.hostMap.Closest(time.Second * 20)
	if err != nil {
		return nil, err
	}

	log.Printf("Found closest host: %+v\n", host)

	client, exists := c.grpcClients[host.Addr]
	if !exists {
		return nil, errors.New("host not found")
	}

	/*
		TODO: add in more realistic client routing logic
	*/

	return client, nil
}

func (c *BlobCacheClient) StoreContent(chunks chan []byte) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, storeContentRequestTimeout)
	defer cancel()

	client, err := c.getGRPCClient()
	if err != nil {
		return "", err
	}

	stream, err := client.StoreContent(ctx)
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

	client, err := c.getGRPCClient()
	if err != nil {
		return err
	}

	resp, err := client.GetState(ctx, &proto.GetStateRequest{})
	if err != nil {
		return err
	}

	log.Println("resp: ", resp)
	return nil
}
