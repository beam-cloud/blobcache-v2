package blobcache

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	proto "github.com/beam-cloud/blobcache/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CacheServiceOpts struct {
	Addr string
}

type CacheService struct {
	proto.UnimplementedBlobCacheServer
	cas    *ContentAddressableStorage
	config BlobCacheConfig
}

func NewCacheService(config BlobCacheConfig) (*CacheService, error) {
	cas, err := NewContentAddressableStorage(config)
	if err != nil {
		return nil, err
	}

	return &CacheService{
		cas:    cas,
		config: config,
	}, nil
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

func (cs *CacheService) StartServer(port uint) error {
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on addr<%s>: %v", addr, err)
	}

	maxMessageSize := cs.config.GRPCMessageSizeBytes
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
	)
	proto.RegisterBlobCacheServer(s, cs)

	log.Printf("Config: %+v\n", cs.config)
	log.Println("Running @", addr)
	go s.Serve(listener)

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
