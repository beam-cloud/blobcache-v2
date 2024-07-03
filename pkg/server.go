package blobcache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	_ "net/http/pprof"
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
	hostname  string
	cas       *ContentAddressableStorage
	cfg       BlobCacheConfig
	tailscale *Tailscale
	metadata  *BlobCacheMetadata
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

	// Mount cache as a FUSE filesystem if blobfs is enabled
	if cfg.BlobFs.Enabled {
		startServer, _, err := Mount(ctx, BlobFsSystemOpts{
			Config:   cfg,
			Metadata: metadata,
		})
		if err != nil {
			return nil, err
		}

		err = startServer()
		if err != nil {
			return nil, err
		}
	}

	tailscale := NewTailscale(hostname, cfg)

	return &CacheService{
		ctx:       ctx,
		hostname:  hostname,
		cas:       cas,
		cfg:       cfg,
		tailscale: tailscale,
		metadata:  metadata,
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

	// Block until a termination signal is received
	terminationChan := make(chan os.Signal, 1)
	signal.Notify(terminationChan, os.Interrupt, syscall.SIGTERM)

	sig := <-terminationChan
	Logger.Infof("Termination signal (%v) received. Shutting down server...", sig)

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

	Logger.Infof("GET - [%s] (offset=%d, length=%d)", req.Hash, req.Offset, req.Length)
	return &proto.GetContentResponse{Content: content, Ok: true}, nil
}

func (cs *CacheService) StoreContent(stream proto.BlobCache_StoreContentServer) error {
	ctx := stream.Context()
	var buffer bytes.Buffer

	fsPath := ""
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if req.FsPath != "" && fsPath == "" {
			fsPath = req.FsPath
		}

		if err != nil {
			Logger.Infof("STORE - error: %v", err)
			return status.Errorf(codes.Unknown, "Received an error: %v", err)
		}

		Logger.Debugf("STORE rx chunk (%d bytes)", len(req.Content))
		if _, err := buffer.Write(req.Content); err != nil {
			Logger.Debugf("STORE - failed to write to buffer: %v", err)
			return status.Errorf(codes.Internal, "Failed to write content to buffer: %v", err)
		}
	}

	content := buffer.Bytes()
	size := len(content)

	Logger.Debugf("STORE rx (%d bytes)", size)

	source := "s3://mock-bucket/key,0-1000" // TODO: replace with real source
	hashBytes := sha256.Sum256(content)
	hash := hex.EncodeToString(hashBytes[:])

	// Store in local in-memory cache
	err := cs.cas.Add(ctx, hash, content, source)
	if err != nil {
		Logger.Infof("STORE - [%s] - %v", hash, err)
		return status.Errorf(codes.Internal, "Failed to add content: %v", err)
	}

	// Store references in blobfs if it's enabled (for disk access to the cached content)
	if cs.cfg.BlobFs.Enabled && fsPath != "" {
		err := cs.metadata.StoreContentInBlobFs(ctx, fsPath, hash, uint64(size))
		if err != nil {
			Logger.Infof("STORE - [%s] unable to store content in blobfs<path=%s> - %v", hash, fsPath, err)
			return status.Errorf(codes.Internal, "Failed to store blobfs reference: %v", err)
		}
	}

	Logger.Infof("STORE - [%s]", hash)
	return stream.SendAndClose(&proto.StoreContentResponse{Hash: hash})
}

func (cs *CacheService) GetState(ctx context.Context, req *proto.GetStateRequest) (*proto.GetStateResponse, error) {
	return &proto.GetStateResponse{Version: BlobCacheVersion}, nil
}
