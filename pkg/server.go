package blobcache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/beam-cloud/blobcache-v2/proto/blobcache_fbs"
	flatbuffers "github.com/google/flatbuffers/go"
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
	blobcache_fbs.UnimplementedBlobCacheServer
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
				Logger.Errorf("Failed to configure content source: %+v\n", err)
				continue
			}

			Logger.Infof("Configured and mounted source: %+v\n", sourceConfig.FilesystemName)
		}
	}

	tailscale := NewTailscale(ctx, hostname, cfg)

	// Start pprof server for performance profiling
	go func() {
		if err := http.ListenAndServe("0.0.0.0:6666", nil); err != nil {
			Logger.Errorf("Failed to start pprof server: %v", err)
		}
	}()

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
		grpc.CustomCodec(flatbuffers.FlatbuffersCodec{}),
	)
	blobcache_fbs.RegisterBlobCacheServer(s, cs)

	Logger.Infof("Running @ %s%s, cfg: %+v\n", cs.hostname, addr, cs.cfg)

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

func (cs *CacheService) GetContent(ctx context.Context, req *blobcache_fbs.GetContentRequest) (*flatbuffers.Builder, error) {
	reqT := req.UnPack()
	content, err := cs.cas.Get(reqT.Hash, reqT.Offset, reqT.Length)
	if err != nil {
		Logger.Debugf("Get - [%s] - %v", reqT.Hash, err)
		b := flatbuffers.NewBuilder(0)
		blobcache_fbs.GetContentResponseStart(b)
		blobcache_fbs.GetContentResponseAddOk(b, false)
		b.Finish(blobcache_fbs.GetContentResponseEnd(b))
		return b, nil
	}

	Logger.Debugf("Get - [%s] (offset=%d, length=%d)", reqT.Hash, reqT.Offset, reqT.Length)
	b := flatbuffers.NewBuilder(0)
	contentOffset := b.CreateByteVector(content)

	blobcache_fbs.GetContentResponseStart(b)
	blobcache_fbs.GetContentResponseAddOk(b, true)
	blobcache_fbs.GetContentResponseAddContent(b, contentOffset)
	b.Finish(blobcache_fbs.GetContentResponseEnd(b))
	return b, nil
}

func (cs *CacheService) store(ctx context.Context, buffer *bytes.Buffer, sourcePath string, sourceOffset int64) (string, error) {
	content := buffer.Bytes()
	size := buffer.Len()

	Logger.Debugf("Store - rx (%d bytes)", size)

	hashBytes := sha256.Sum256(content)
	hash := hex.EncodeToString(hashBytes[:])

	// Store in local in-memory cache
	err := cs.cas.Add(ctx, hash, content, sourcePath, sourceOffset)
	if err != nil {
		Logger.Infof("Store - [%s] - %v", hash, err)
		return "", status.Errorf(codes.Internal, "Failed to add content: %v", err)
	}

	// Store references in blobfs if it's enabled (for disk access to the cached content)
	if cs.cfg.BlobFs.Enabled && sourcePath != "" {
		err := cs.metadata.StoreContentInBlobFs(ctx, sourcePath, hash, uint64(size))
		if err != nil {
			Logger.Infof("Store - [%s] unable to store content in blobfs<path=%s> - %v", hash, sourcePath, err)
			return "", status.Errorf(codes.Internal, "Failed to store blobfs reference: %v", err)
		}
	}

	Logger.Infof("Store - [%s]", hash)
	content = nil
	return hash, nil
}

func (cs *CacheService) StoreContent(stream blobcache_fbs.BlobCache_StoreContentServer) error {
	ctx := stream.Context()
	var buffer bytes.Buffer

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			Logger.Infof("Store - error: %v", err)
			return status.Errorf(codes.Unknown, "Received an error: %v", err)
		}
		reqBytes := req.ContentBytes()

		// Convert received bytes to FlatBuffers request
		Logger.Debugf("Store - rx chunk (%d bytes)", len(reqBytes))

		if _, err := buffer.Write(reqBytes); err != nil {
			Logger.Debugf("Store - failed to write to buffer: %v", err)
			return status.Errorf(codes.Internal, "Failed to write content to buffer: %v", err)
		}
	}

	hash, err := cs.store(ctx, &buffer, "", 0)
	if err != nil {
		return err
	}

	buffer.Reset()

	// Build response using FlatBuffers
	b := flatbuffers.NewBuilder(0)
	hashOffset := b.CreateString(hash)
	blobcache_fbs.StoreContentResponseStart(b)
	blobcache_fbs.StoreContentResponseAddHash(b, hashOffset)
	b.Finish(blobcache_fbs.StoreContentResponseEnd(b))
	return stream.SendAndClose(b)
}

func (cs *CacheService) usagePct() float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memoryUsage := float64(memStats.Alloc) / (1024 * 1024)
	return memoryUsage / float64(cs.cas.maxCacheSizeMb)
}

func (cs *CacheService) GetState(ctx context.Context, req *blobcache_fbs.GetStateRequest) (*flatbuffers.Builder, error) {
	b := flatbuffers.NewBuilder(0)
	version := b.CreateString(BlobCacheVersion)
	privateIpAddr := b.CreateString(cs.privateIpAddr)

	blobcache_fbs.GetStateResponseStart(b)
	blobcache_fbs.GetStateResponseAddVersion(b, version)
	blobcache_fbs.GetStateResponseAddPrivateIpAddr(b, privateIpAddr)
	blobcache_fbs.GetStateResponseAddCapacityUsagePct(b, float32(cs.usagePct()))
	b.Finish(blobcache_fbs.GetStateResponseEnd(b))
	return b, nil
}

func (cs *CacheService) StoreContentFromSource(ctx context.Context, req *blobcache_fbs.StoreContentFromSourceRequest) (*flatbuffers.Builder, error) {
	reqT := req.UnPack()
	localPath := filepath.Join("/", reqT.SourcePath)

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		Logger.Infof("StoreFromContent - source not found: %v", err)
		b := flatbuffers.NewBuilder(0)
		blobcache_fbs.StoreContentFromSourceResponseStart(b)
		blobcache_fbs.StoreContentFromSourceResponseAddOk(b, false)
		b.Finish(blobcache_fbs.StoreContentFromSourceResponseEnd(b))
		return b, status.Errorf(codes.NotFound, "File does not exist: %s", localPath)
	}

	file, err := os.Open(localPath)
	if err != nil {
		Logger.Infof("StoreFromContent - error reading source: %v", err)
		b := flatbuffers.NewBuilder(0)
		blobcache_fbs.StoreContentFromSourceResponseStart(b)
		blobcache_fbs.StoreContentFromSourceResponseAddOk(b, false)
		b.Finish(blobcache_fbs.StoreContentFromSourceResponseEnd(b))
		return b, status.Errorf(codes.Internal, "Failed to open file: %v", err)
	}
	defer file.Close()

	var buffer bytes.Buffer

	if _, err := io.Copy(&buffer, file); err != nil {
		Logger.Infof("StoreFromContent - error copying source: %v", err)
		b := flatbuffers.NewBuilder(0)
		blobcache_fbs.StoreContentFromSourceResponseStart(b)
		blobcache_fbs.StoreContentFromSourceResponseAddOk(b, false)
		b.Finish(blobcache_fbs.StoreContentFromSourceResponseEnd(b))
		return b, nil
	}

	hash, err := cs.store(ctx, &buffer, localPath, reqT.SourceOffset)
	if err != nil {
		Logger.Infof("StoreFromContent - error storing data in cache: %v", err)
		b := flatbuffers.NewBuilder(0)
		blobcache_fbs.StoreContentFromSourceResponseStart(b)
		blobcache_fbs.StoreContentFromSourceResponseAddOk(b, false)
		b.Finish(blobcache_fbs.StoreContentFromSourceResponseEnd(b))
		return b, err
	}

	b := flatbuffers.NewBuilder(0)
	hashOffset := b.CreateString(hash)
	blobcache_fbs.StoreContentFromSourceResponseStart(b)
	blobcache_fbs.StoreContentFromSourceResponseAddOk(b, true)
	blobcache_fbs.StoreContentFromSourceResponseAddHash(b, hashOffset)
	b.Finish(blobcache_fbs.StoreContentFromSourceResponseEnd(b))
	return b, nil
}
