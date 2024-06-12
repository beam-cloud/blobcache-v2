package blobcache

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	proto "github.com/beam-cloud/blobcache/proto"
	"github.com/google/uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CacheServiceOpts struct {
	Addr string
}

type CacheService struct {
	proto.UnimplementedBlobCacheServer
	hostname  string
	cas       *ContentAddressableStorage
	cfg       BlobCacheConfig
	tailscale *Tailscale
	metadata  *BlobCacheMetadata
	discovery *DiscoveryClient
}

func NewCacheService(cfg BlobCacheConfig) (*CacheService, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheServicePrefix, uuid.New().String()[:6])
	log.Printf("Hostname is %s\n", hostname)

	cas, err := NewContentAddressableStorage(cfg)
	if err != nil {
		return nil, err
	}

	metadata, err := NewBlobCacheMetadata(cfg.Metadata)
	if err != nil {
		return nil, err
	}

	tailscale := NewTailscale(hostname, cfg)
	return &CacheService{
		hostname:  hostname,
		cas:       cas,
		cfg:       cfg,
		tailscale: tailscale,
		metadata:  metadata,
		discovery: NewDiscoveryClient(cfg, tailscale),
	}, nil
}

func (cs *CacheService) StartServer(port uint) error {
	addr := fmt.Sprintf(":%d", port)

	server := cs.tailscale.GetOrCreateServer()
	ln, err := server.Listen("tcp", addr)
	if err != nil {
		return err
	}

	maxMessageSize := cs.cfg.GRPCMessageSizeBytes
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
	)
	proto.RegisterBlobCacheServer(s, cs)

	log.Printf("Running @ %s, cfg: %+v\n", addr, cs.cfg)
	go s.Serve(ln)

	go cs.discovery.Start(context.TODO())

	// Create a channel to receive termination signals
	terminationSignal := make(chan os.Signal, 1)
	signal.Notify(terminationSignal, os.Interrupt, syscall.SIGTERM)

	// Block until a termination signal is received
	<-terminationSignal
	log.Println("Termination signal received. Shutting down...")

	// Close in-memory cache
	s.GracefulStop()
	cs.cas.Cleanup()
	return nil
}

func (cs *CacheService) GetState(ctx context.Context, req *proto.GetStateRequest) (*proto.GetStateResponse, error) {
	return &proto.GetStateResponse{Version: BlobCacheVersion}, nil
}

func (cs *CacheService) GetContent(ctx context.Context, req *proto.GetContentRequest) (*proto.GetContentResponse, error) {
	content, err := cs.cas.Get(req.Hash, req.Offset, req.Length)
	if err != nil {
		return nil, err
	}
	return &proto.GetContentResponse{Content: content}, nil
}

func (cs *CacheService) StoreContent(stream proto.BlobCache_StoreContentServer) error {
	var content []byte

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return status.Errorf(codes.Unknown, "Received an error: %v", err)
		}

		content = append(content, req.Content...)
	}

	hash, err := cs.cas.Add(content)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to add content: %v", err)
	}

	return stream.SendAndClose(&proto.StoreContentResponse{Hash: hash})
}
