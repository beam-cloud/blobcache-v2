package blobcache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"github.com/google/uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	writeBufferSizeBytes      int   = 128 * 1024
	getContentStreamChunkSize int64 = 16 * 1024 * 1024 // 16MB
)

type CacheServiceOpts struct {
	Addr string
}

type CacheService struct {
	ctx context.Context
	proto.UnimplementedBlobCacheServer
	hostname      string
	privateIpAddr string
	cas           *ContentAddressableStorage
	cfg           BlobCacheConfig
	tailscale     *Tailscale
	metadata      *BlobCacheMetadata
}

func NewCacheService(ctx context.Context, cfg BlobCacheConfig) (*CacheService, error) {
	hostname := fmt.Sprintf("%s-%s", BlobCacheHostPrefix, uuid.New().String()[:6])

	// If HostStorageCapacityThresholdPct is not set, make sure a sensible default is set
	if cfg.HostStorageCapacityThresholdPct <= 0 {
		cfg.HostStorageCapacityThresholdPct = defaultHostStorageCapacityThresholdPct
	}

	currentHost := &BlobCacheHost{
		Addr: fmt.Sprintf("%s.%s:%d", hostname, cfg.Tailscale.HostName, cfg.Port),
		RTT:  0,
	}

	metadata, err := NewBlobCacheMetadata(cfg.Metadata)
	if err != nil {
		return nil, err
	}

	privateIpAddr, _ := GetPrivateIpAddr()
	if privateIpAddr != "" {
		Logger.Infof("Discovered private ip address: %s", privateIpAddr)
	}
	currentHost.PrivateAddr = fmt.Sprintf("%s:%d", privateIpAddr, cfg.Port)
	currentHost.CapacityUsagePct = 0

	cas, err := NewContentAddressableStorage(ctx, currentHost, metadata, cfg)
	if err != nil {
		return nil, err
	}

	// Mount cache as a FUSE filesystem if blobfs is enabled
	if cfg.BlobFs.Enabled {
		for _, sourceConfig := range cfg.BlobFs.Sources {
			_, err := NewSource(sourceConfig)
			if err != nil {
				Logger.Errorf("Failed to configure content source: %+v", err)
				continue
			}

			Logger.Infof("Configured and mounted source: %+v", sourceConfig.FilesystemName)
		}
	}

	tailscale := NewTailscale(ctx, hostname, cfg)

	cs := &CacheService{
		ctx:           ctx,
		hostname:      hostname,
		cas:           cas,
		cfg:           cfg,
		tailscale:     tailscale,
		metadata:      metadata,
		privateIpAddr: privateIpAddr,
	}

	go cs.HostKeepAlive()

	return cs, nil
}

func (cs *CacheService) HostKeepAlive() {
	err := cs.metadata.SetHostKeepAlive(cs.ctx, cs.cas.currentHost)
	if err != nil {
		Logger.Warnf("Failed to set host keepalive: %v", err)
	}

	err = cs.metadata.AddHostToIndex(cs.ctx, cs.cas.currentHost)
	if err != nil {
		Logger.Warnf("Failed to add host to index: %v", err)
	}

	ticker := time.NewTicker(time.Duration(defaultHostKeepAliveIntervalS) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			cs.cas.currentHost.PrivateAddr = fmt.Sprintf("%s:%d", cs.privateIpAddr, cs.cfg.Port)
			cs.cas.currentHost.CapacityUsagePct = cs.usagePct()

			cs.metadata.AddHostToIndex(cs.ctx, cs.cas.currentHost)
			cs.metadata.SetHostKeepAlive(cs.ctx, cs.cas.currentHost)
		}
	}
}

func (cs *CacheService) StartServer(port uint) error {
	addr := fmt.Sprintf(":%d", port)

	server, err := cs.tailscale.GetOrCreateServer()
	if err != nil {
		return err
	}

	// Bind both to tailnet and locally
	tailscaleListener, err := server.Listen("tcp", addr)
	if err != nil {
		return err
	}

	localListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	maxMessageSize := cs.cfg.GRPCMessageSizeBytes
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
		grpc.WriteBufferSize(writeBufferSizeBytes),
		grpc.NumStreamWorkers(uint32(runtime.NumCPU())),
	)
	proto.RegisterBlobCacheServer(s, cs)

	Logger.Infof("Running @ %s%s, cfg: %+v", cs.hostname, addr, cs.cfg)

	go s.Serve(localListener)
	go s.Serve(tailscaleListener)

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
	dst := make([]byte, req.Length)
	n, err := cs.cas.Get(req.Hash, req.Offset, req.Length, dst)
	if err != nil {
		Logger.Debugf("Get - [%s] - %v", req.Hash, err)
		return &proto.GetContentResponse{Content: nil, Ok: false}, nil
	}

	Logger.Debugf("Get[OK] - [%s] (offset=%d, length=%d)", req.Hash, req.Offset, req.Length)
	return &proto.GetContentResponse{Content: dst[:n], Ok: true}, nil
}

func (cs *CacheService) GetContentStream(req *proto.GetContentRequest, stream proto.BlobCache_GetContentStreamServer) error {
	const chunkSize = getContentStreamChunkSize
	offset := req.Offset
	remainingLength := req.Length
	Logger.Infof("GetContentStream[ACK] - [%s] - offset=%d, length=%d, %d bytes", req.Hash, offset, req.Length, remainingLength)

	dst := make([]byte, chunkSize)
	for remainingLength > 0 {
		currentChunkSize := chunkSize
		if remainingLength < int64(chunkSize) {
			currentChunkSize = remainingLength
		}

		n, err := cs.cas.Get(req.Hash, offset, currentChunkSize, dst)
		if err != nil {
			Logger.Debugf("GetContentStream - [%s] - %v", req.Hash, err)
			return status.Errorf(codes.NotFound, "Content not found: %v", err)
		}

		if n == 0 {
			break
		}

		Logger.Debugf("GetContentStream[TX] - [%s] - %d bytes", req.Hash, n)
		if err := stream.Send(&proto.GetContentResponse{
			Ok:      true,
			Content: dst[:n],
		}); err != nil {
			return status.Errorf(codes.Internal, "Failed to send content chunk: %v", err)
		}

		// Break if this is the last chunk
		if n < currentChunkSize {
			break
		}

		offset += int64(n)
		remainingLength -= int64(n)
	}

	return nil
}

func (cs *CacheService) store(ctx context.Context, buffer *bytes.Buffer, sourcePath string, sourceOffset int64) (string, error) {
	content := buffer.Bytes()
	size := buffer.Len()

	Logger.Infof("Store[ACK] (%d bytes)", size)

	hashBytes := sha256.Sum256(content)
	hash := hex.EncodeToString(hashBytes[:])

	// Store in local in-memory cache
	err := cs.cas.Add(ctx, hash, content, sourcePath, sourceOffset)
	if err != nil {
		Logger.Infof("Store[ERR] - [%s] - %v", hash, err)
		return "", status.Errorf(codes.Internal, "Failed to add content: %v", err)
	}

	// Store references in blobfs if it's enabled (for disk access to the cached content)
	if cs.cfg.BlobFs.Enabled && sourcePath != "" {
		err := cs.metadata.StoreContentInBlobFs(ctx, sourcePath, hash, uint64(size))
		if err != nil {
			Logger.Infof("Store[ERR] - [%s] unable to store content in blobfs<path=%s> - %v", hash, sourcePath, err)
			return "", status.Errorf(codes.Internal, "Failed to store blobfs reference: %v", err)
		}
	}

	Logger.Infof("Store[OK] - [%s]", hash)
	content = nil
	return hash, nil
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
			Logger.Infof("Store[ERR] - error: %v", err)
			return status.Errorf(codes.Unknown, "Received an error: %v", err)
		}

		Logger.Debugf("Store[RX] - chunk (%d bytes)", len(req.Content))
		if _, err := buffer.Write(req.Content); err != nil {
			Logger.Debugf("Store[ERR] - failed to write to buffer: %v", err)
			return status.Errorf(codes.Internal, "Failed to write content to buffer: %v", err)
		}
	}

	hash, err := cs.store(ctx, &buffer, "", 0)
	if err != nil {
		return err
	}

	buffer.Reset()
	return stream.SendAndClose(&proto.StoreContentResponse{Hash: hash})
}

func (cs *CacheService) usagePct() float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryUsage := float64(memStats.Alloc) / (1024 * 1024)
	return memoryUsage / float64(cs.cas.maxCacheSizeMb)
}

func (cs *CacheService) GetState(ctx context.Context, req *proto.GetStateRequest) (*proto.GetStateResponse, error) {
	return &proto.GetStateResponse{
		Version:          BlobCacheVersion,
		PrivateIpAddr:    cs.privateIpAddr,
		CapacityUsagePct: float32(cs.usagePct()),
	}, nil
}

func (cs *CacheService) StoreContentFromSource(ctx context.Context, req *proto.StoreContentFromSourceRequest) (*proto.StoreContentFromSourceResponse, error) {
	localPath := filepath.Join("/", req.SourcePath)
	Logger.Infof("StoreFromContent[ACK] - [%s]", localPath)

	// Check if the file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		Logger.Infof("StoreFromContent[ERR] - source not found: %v", err)
		return &proto.StoreContentFromSourceResponse{Ok: false}, status.Errorf(codes.NotFound, "File does not exist: %s", localPath)
	}

	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		Logger.Infof("StoreFromContent[ERR] - error reading source: %v", err)
		return &proto.StoreContentFromSourceResponse{Ok: false}, status.Errorf(codes.Internal, "Failed to open file: %v", err)
	}
	defer file.Close()

	var buffer bytes.Buffer

	if _, err := io.Copy(&buffer, file); err != nil {
		Logger.Infof("StoreFromContent[ERR] - error copying source: %v", err)
		return &proto.StoreContentFromSourceResponse{Ok: false}, nil
	}

	// Store the content
	hash, err := cs.store(ctx, &buffer, localPath, req.SourceOffset)
	if err != nil {
		Logger.Infof("StoreFromContent[ERR] - error storing data in cache: %v", err)
		return &proto.StoreContentFromSourceResponse{Ok: false}, err
	}

	buffer.Reset()
	Logger.Infof("StoreFromContent[OK] - [%s]", hash)

	// HOTFIX: Manually trigger garbage collection
	go runtime.GC()

	return &proto.StoreContentFromSourceResponse{Ok: true, Hash: hash}, nil
}
