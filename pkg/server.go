package blobcache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"github.com/google/uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CacheServiceOpts struct {
	Addr string
}

type CacheService struct {
	ctx context.Context
	proto.UnimplementedBlobCacheServer
	hostname        string
	cas             *ContentAddressableStorage
	cfg             BlobCacheConfig
	tailscale       *Tailscale
	metadata        *BlobCacheMetadata
	discoveryClient *DiscoveryClient
	hostMap         *HostMap
}

func NewCacheService(ctx context.Context, cfg BlobCacheConfig) (*CacheService, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheHostPrefix, uuid.New().String()[:6])
	currentHost := &BlobCacheHost{
		Addr: fmt.Sprintf("%s.%s:%d", hostname, cfg.Tailscale.HostName, cfg.Port),
		RTT:  0,
	}

	metadata, err := NewBlobCacheMetadata(cfg.Metadata)
	if err != nil {
		return nil, err
	}

	cas, err := NewContentAddressableStorage(ctx, currentHost, metadata, cfg)
	if err != nil {
		return nil, err
	}

	hostMap := NewHostMap(nil, nil)
	tailscale := NewTailscale(hostname, cfg)

	return &CacheService{
		ctx:             ctx,
		hostname:        hostname,
		cas:             cas,
		cfg:             cfg,
		tailscale:       tailscale,
		metadata:        metadata,
		discoveryClient: NewDiscoveryClient(cfg, tailscale, hostMap),
		hostMap:         hostMap,
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

	Logger.Infof("Running @ %s%s, cfg: %+v\n", cs.hostname, addr, cs.cfg)

	go s.Serve(ln)
	go cs.discoveryClient.StartInBackground(context.TODO())

	// Block until a termination signal is received
	terminationChan := make(chan os.Signal, 1)
	signal.Notify(terminationChan, os.Interrupt, syscall.SIGTERM)
	<-terminationChan

	Logger.Info("Termination signal received. Shutting down server...")

	// Close in-memory cache
	s.GracefulStop()
	cs.cas.Cleanup()
	return nil
}

func (cs *CacheService) GetContent(ctx context.Context, req *proto.GetContentRequest) (*proto.GetContentResponse, error) {
	content, err := cs.cas.Get(req.Hash, req.Offset, req.Length)
	if err != nil {
		Logger.Debugf("GET - [%s] - %v", req.Hash, err)
		return &proto.GetContentResponse{Content: nil, Ok: false}, nil
	}

	Logger.Infof("GET - [%s]", req.Hash)
	return &proto.GetContentResponse{Content: content, Ok: true}, nil
}

func (cs *CacheService) StoreContent(stream proto.BlobCache_StoreContentServer) error {
	ctx := stream.Context()
	var buffer bytes.Buffer

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			Logger.Infof("STORE - error occurred: %v", err)
			return status.Errorf(codes.Unknown, "Received an error: %v", err)
		}

		Logger.Debugf("STORE rx chunk - %d bytes", len(req.Content))
		if _, err := buffer.Write(req.Content); err != nil {
			Logger.Debugf("STORE - failed to write to buffer: %v", err)
			return status.Errorf(codes.Internal, "Failed to write content to buffer: %v", err)
		}
	}

	content := buffer.Bytes()
	Logger.Debugf("STORE Received %d bytes", len(content))

	hash, err := cs.cas.Add(ctx, content, "s3://mock-bucket/key,0-1000")
	if err != nil {
		Logger.Infof("STORE - [%s] - %v", hash, err)
		return status.Errorf(codes.Internal, "Failed to add content: %v", err)
	}

	Logger.Infof("STORE - [%s]", hash)
	return stream.SendAndClose(&proto.StoreContentResponse{Hash: hash})
}

func (cs *CacheService) GetState(ctx context.Context, req *proto.GetStateRequest) (*proto.GetStateResponse, error) {
	return &proto.GetStateResponse{Version: BlobCacheVersion}, nil
}
