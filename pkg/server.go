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
	"strings"
	"sync"
	"syscall"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"github.com/djherbis/atime"
	"github.com/google/uuid"
	"github.com/hanwen/go-fuse/v2/fuse"

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
	ctx  context.Context
	mode BlobCacheServerMode
	proto.UnimplementedBlobCacheServer
	hostId        string
	locality      string
	privateIpAddr string
	publicIpAddr  string
	cas           *ContentAddressableStorage
	serverConfig  BlobCacheServerConfig
	globalConfig  BlobCacheGlobalConfig
	coordinator   CoordinatorClient
	s3ClientCache sync.Map
}

func NewCacheService(ctx context.Context, cfg BlobCacheConfig, locality string) (*CacheService, error) {
	currentHost := &BlobCacheHost{
		RTT: 0,
	}

	var coordinator CoordinatorClient
	var err error = nil
	switch cfg.Server.Mode {
	case BlobCacheServerModeCoordinator:
		coordinator, err = NewCoordinatorClientLocal(cfg.Global, cfg.Server)
	default:
		coordinator, err = NewCoordinatorClientRemote(cfg.Global, cfg.Client.Token)
		if err != nil {
			return nil, err
		}

		regionConfig, err := coordinator.GetRegionConfig(ctx, locality)
		if err != nil {
			Logger.Infof("No region-specific config found for locality %s, using current config", locality)
		} else {
			cfg.Server = regionConfig
		}
	}
	if err != nil {
		return nil, err
	}

	// Create the disk cache directory if it doesn't exist
	err = os.MkdirAll(cfg.Server.DiskCacheDir, 0755)
	if err != nil {
		return nil, err
	}

	hostId := getHostId(cfg.Server)
	Logger.Infof("Server<%s> started in %s mode", hostId, cfg.Server.Mode)

	publicIpAddr, _ := GetPublicIpAddr()
	if publicIpAddr != "" {
		Logger.Infof("Discovered public ip address: %s", publicIpAddr)
	}

	privateIpAddr, _ := GetPrivateIpAddr()
	if privateIpAddr != "" {
		Logger.Infof("Discovered private ip address: %s", privateIpAddr)
	}

	currentHost.HostId = hostId
	currentHost.Addr = fmt.Sprintf("%s:%d", publicIpAddr, cfg.Global.ServerPort)
	currentHost.PrivateAddr = fmt.Sprintf("%s:%d", privateIpAddr, cfg.Global.ServerPort)
	currentHost.CapacityUsagePct = 0

	cas, err := NewContentAddressableStorage(ctx, currentHost, locality, coordinator, cfg)
	if err != nil {
		return nil, err
	}

	for _, sourceConfig := range cfg.Server.Sources {
		_, err := NewSource(sourceConfig)
		if err != nil {
			Logger.Errorf("Failed to configure content source: %+v", err)
			continue
		}

		Logger.Infof("Configured and mounted source: %+v", sourceConfig.FilesystemName)
	}

	cs := &CacheService{
		ctx:           ctx,
		mode:          cfg.Server.Mode,
		hostId:        hostId,
		locality:      locality,
		cas:           cas,
		serverConfig:  cfg.Server,
		globalConfig:  cfg.Global,
		coordinator:   coordinator,
		privateIpAddr: privateIpAddr,
		publicIpAddr:  publicIpAddr,
		s3ClientCache: sync.Map{},
	}

	go cs.HostKeepAlive()
	return cs, nil
}

func getHostId(serverConfig BlobCacheServerConfig) string {
	filePath := filepath.Join(serverConfig.DiskCacheDir, "HOST_ID")

	hostId := ""
	if content, err := os.ReadFile(filePath); err == nil {
		hostId = strings.TrimSpace(string(content))
	} else {
		hostId = fmt.Sprintf("%s-%s", BlobCacheHostPrefix, uuid.New().String()[:6])
		os.WriteFile(filePath, []byte(hostId), 0644)
	}

	return hostId
}

func (cs *CacheService) HostKeepAlive() {
	err := cs.coordinator.SetHostKeepAlive(cs.ctx, cs.locality, cs.cas.currentHost)
	if err != nil {
		Logger.Warnf("Failed to set host keepalive: %v", err)
	}

	err = cs.coordinator.AddHostToIndex(cs.ctx, cs.locality, cs.cas.currentHost)
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
			cs.cas.currentHost.PrivateAddr = fmt.Sprintf("%s:%d", cs.privateIpAddr, cs.globalConfig.ServerPort)
			cs.cas.currentHost.CapacityUsagePct = cs.usagePct()

			cs.coordinator.AddHostToIndex(cs.ctx, cs.locality, cs.cas.currentHost)
			cs.coordinator.SetHostKeepAlive(cs.ctx, cs.locality, cs.cas.currentHost)
		}
	}
}

func (cs *CacheService) StartServer(port uint) error {
	addr := fmt.Sprintf(":%d", port)

	localListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	maxMessageSize := cs.globalConfig.GRPCMessageSizeBytes
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
		grpc.WriteBufferSize(writeBufferSizeBytes),
		grpc.NumStreamWorkers(uint32(runtime.NumCPU())),
	)
	proto.RegisterBlobCacheServer(s, cs)

	Logger.Infof("Running %s@%s, cfg: %+v", cs.hostId, addr, cs.serverConfig)

	go s.Serve(localListener)

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

func (cs *CacheService) HasContent(ctx context.Context, req *proto.HasContentRequest) (*proto.HasContentResponse, error) {
	exists := cs.cas.Exists(req.Hash)
	return &proto.HasContentResponse{Exists: exists, Ok: true}, nil
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

func (cs *CacheService) store(ctx context.Context, buffer *bytes.Buffer) (string, error) {
	content := buffer.Bytes()
	size := buffer.Len()

	Logger.Infof("Store[ACK] (%d bytes)", size)

	hashBytes := sha256.Sum256(content)
	hash := hex.EncodeToString(hashBytes[:])

	// Store in local in-memory cache
	err := cs.cas.Add(ctx, hash, content)
	if err != nil {
		Logger.Infof("Store[ERR] - [%s] - %v", hash, err)
		return "", status.Errorf(codes.Internal, "Failed to add content: %v", err)
	}

	Logger.Infof("Store[OK] - [%s]", hash)
	content = nil
	return hash, nil
}

func (cs *CacheService) StoreContentInBlobFs(ctx context.Context, path string, hash string, size uint64) error {
	path = filepath.Join("/", filepath.Clean(path))
	parts := strings.Split(path, string(filepath.Separator))

	rootParentId := GenerateFsID("/")

	// Iterate over the components and construct the path hierarchy
	currentPath := "/"
	previousParentId := rootParentId // start with the root ID
	for i, part := range parts {
		if i == 0 && part == "" {
			continue // Skip the empty part for root
		}

		if currentPath == "/" {
			currentPath = filepath.Join("/", part)
		} else {
			currentPath = filepath.Join(currentPath, part)
		}

		currentNodeId := GenerateFsID(currentPath)
		inode, err := SHA1StringToUint64(currentNodeId)
		if err != nil {
			return err
		}

		// Initialize default metadata
		now := time.Now()
		nowSec := uint64(now.Unix())
		nowNsec := uint32(now.Nanosecond())
		metadata := &BlobFsMetadata{
			PID:       previousParentId,
			ID:        currentNodeId,
			Name:      part,
			Path:      currentPath,
			Ino:       inode,
			Mode:      fuse.S_IFDIR | 0755,
			Atime:     nowSec,
			Mtime:     nowSec,
			Ctime:     nowSec,
			Atimensec: nowNsec,
			Mtimensec: nowNsec,
			Ctimensec: nowNsec,
		}

		// If currentPath matches the input path, use the actual file info
		if currentPath == path {
			fileInfo, err := os.Stat(currentPath)
			if err != nil {
				return err
			}

			// Update metadata fields with actual file info values
			modTime := fileInfo.ModTime()
			accessTime := atime.Get(fileInfo)
			metadata.Mode = uint32(fileInfo.Mode())
			metadata.Atime = uint64(accessTime.Unix())
			metadata.Atimensec = uint32(accessTime.Nanosecond())
			metadata.Mtime = uint64(modTime.Unix())
			metadata.Mtimensec = uint32(modTime.Nanosecond())

			// Since we cannot get Ctime in a platform-independent way, set it to ModTime
			metadata.Ctime = uint64(modTime.Unix())
			metadata.Ctimensec = uint32(modTime.Nanosecond())

			metadata.Size = uint64(fileInfo.Size())
			if fileInfo.IsDir() {
				metadata.Hash = GenerateFsID(currentPath)
				metadata.Size = 0
			} else {
				metadata.Hash = hash
				metadata.Size = size
			}
		}

		// Set metadata
		err = cs.coordinator.SetFsNode(ctx, currentNodeId, metadata)
		if err != nil {
			return err
		}

		// Add the current node as a child of the previous node
		err = cs.coordinator.AddFsNodeChild(ctx, previousParentId, currentNodeId)
		if err != nil {
			return err
		}

		previousParentId = currentNodeId
	}

	return nil
}

func (cs *CacheService) StoreContent(stream proto.BlobCache_StoreContentServer) error {
	ctx := stream.Context()
	var buffer bytes.Buffer

	Logger.Infof("StoreContent[ACK]")

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

	hash, err := cs.store(ctx, &buffer)
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

func (cs *CacheService) cacheSourceFromLocalPath(localPath string, buffer *bytes.Buffer) error {
	// Check if the file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		Logger.Infof("StoreFromContent[ERR] - source not found: %v", err)
		return err
	}

	// Open the file
	file, err := os.Open(localPath)
	if err != nil {
		Logger.Infof("StoreFromContent[ERR] - error reading source: %v", err)
		return err
	}
	defer file.Close()

	if _, err := io.Copy(buffer, file); err != nil {
		Logger.Infof("StoreFromContent[ERR] - error copying source: %v", err)
		return err
	}

	return nil
}

func (cs *CacheService) cacheSourceFromS3(source *proto.CacheSource, buffer *bytes.Buffer) error {
	key := fmt.Sprintf("%s/%s/%s/%s", source.EndpointUrl, source.Region, source.BucketName, source.Path)

	var s3Client *S3Client
	if cachedS3Client, ok := cs.s3ClientCache.Load(key); ok {
		s3Client = cachedS3Client.(*S3Client)
	} else {
		s3Client, err := NewS3Client(cs.ctx, struct {
			BucketName  string
			Path        string
			Region      string
			EndpointURL string
			AccessKey   string
			SecretKey   string
		}{
			BucketName:  source.BucketName,
			Path:        source.Path,
			Region:      source.Region,
			EndpointURL: source.EndpointUrl,
			AccessKey:   source.AccessKey,
			SecretKey:   source.SecretKey,
		})
		if err != nil {
			return err
		}

		cs.s3ClientCache.Store(key, s3Client)
	}

	err := s3Client.DownloadIntoBuffer(cs.ctx, source.Path, buffer)
	if err != nil {
		return err
	}

	return nil
}

func (cs *CacheService) StoreContentFromSource(ctx context.Context, req *proto.StoreContentFromSourceRequest) (*proto.StoreContentFromSourceResponse, error) {
	localPath := filepath.Join("/", req.Source.Path)
	Logger.Infof("StoreFromContent[ACK] - [%s]", localPath)

	var buffer bytes.Buffer
	if req.Source.BucketName == "" {
		err := cs.cacheSourceFromLocalPath(localPath, &buffer)
		if err != nil {
			return &proto.StoreContentFromSourceResponse{Ok: false, ErrorMsg: err.Error()}, err
		}
	} else {
		err := cs.cacheSourceFromS3(req.Source, &buffer)
		if err != nil {
			return &proto.StoreContentFromSourceResponse{Ok: false, ErrorMsg: err.Error()}, err
		}
	}

	// Store the content
	hash, err := cs.store(ctx, &buffer)
	if err != nil {
		Logger.Infof("StoreFromContent[ERR] - error storing data in cache: %v", err)
		return &proto.StoreContentFromSourceResponse{Ok: false, ErrorMsg: err.Error()}, nil
	}

	// Store references in blobfs if it's enabled (for disk access to the cached content)
	// This is unnecessary for workspace storage, but still required for CLIP to lazy load content from cache
	// and volume caching + juicefs to work
	if cs.coordinator != nil && req.Source.BucketName == "" {
		err := cs.StoreContentInBlobFs(ctx, localPath, hash, uint64(buffer.Len()))
		if err != nil {
			Logger.Infof("Store[ERR] - [%s] unable to store content in blobfs<path=%s> - %v", hash, localPath, err)
			return &proto.StoreContentFromSourceResponse{Ok: false, ErrorMsg: err.Error()}, nil
		}
	}

	buffer.Reset()
	Logger.Infof("StoreFromContent[OK] - [%s]", hash)

	// HOTFIX: Manually trigger garbage collection
	go runtime.GC()

	return &proto.StoreContentFromSourceResponse{Ok: true, Hash: hash}, nil
}

func (cs *CacheService) StoreContentFromSourceWithLock(ctx context.Context, req *proto.StoreContentFromSourceRequest) (*proto.StoreContentFromSourceWithLockResponse, error) {
	sourcePath := req.Source.Path
	if err := cs.coordinator.SetStoreFromContentLock(ctx, cs.locality, sourcePath); err != nil {
		return &proto.StoreContentFromSourceWithLockResponse{Ok: false, FailedToAcquireLock: true, ErrorMsg: err.Error()}, nil
	}

	storeContext, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-storeContext.Done():
				return
			case <-ticker.C:
				Logger.Infof("StoreContentFromSourceWithLock[REFRESH] - [%s]", sourcePath)
				cs.coordinator.RefreshStoreFromContentLock(ctx, cs.locality, sourcePath)
			}
		}
	}()

	storeContentFromSourceResp, err := cs.StoreContentFromSource(storeContext, req)
	if err != nil {
		return &proto.StoreContentFromSourceWithLockResponse{Hash: "", Ok: false, ErrorMsg: err.Error()}, nil
	}

	if err := cs.coordinator.RemoveStoreFromContentLock(ctx, cs.locality, sourcePath); err != nil {
		Logger.Errorf("StoreContentFromSourceWithLock[ERR] - error removing lock: %v", err)
	}

	return &proto.StoreContentFromSourceWithLockResponse{Hash: storeContentFromSourceResp.Hash, Ok: true}, nil
}
