package blobcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/djherbis/atime"
	"github.com/hanwen/go-fuse/v2/fuse"
	redis "github.com/redis/go-redis/v9"
)

const (
	storeFromContentLockTtlS = 5
)

type BlobCacheMetadata struct {
	rdb  *RedisClient
	lock *RedisLock
}

func NewBlobCacheMetadata(cfg MetadataConfig) (*BlobCacheMetadata, error) {
	redisMode := RedisModeSingle
	if cfg.RedisMode != "" {
		redisMode = cfg.RedisMode
	}

	rdb, err := NewRedisClient(RedisConfig{
		Addrs:              []string{cfg.RedisAddr},
		Mode:               redisMode,
		Password:           cfg.RedisPasswd,
		SentinelPassword:   cfg.RedisPasswd,
		EnableTLS:          cfg.RedisTLSEnabled,
		MasterName:         cfg.RedisMasterName,
		InsecureSkipVerify: true, // HOTFIX: tailscale certs don't match in-cluster certs
	})
	if err != nil {
		return nil, err
	}

	lock := NewRedisLock(rdb)
	return &BlobCacheMetadata{
		rdb:  rdb,
		lock: lock,
	}, nil
}

func (m *BlobCacheMetadata) SetClientLock(ctx context.Context, clientId, hash string) error {
	err := m.lock.Acquire(ctx, MetadataKeys.MetadataClientLock(clientId, hash), RedisLockOptions{TtlS: 300, Retries: 0})
	if err != nil {
		return err
	}

	return nil
}

func (m *BlobCacheMetadata) RemoveClientLock(ctx context.Context, clientId, hash string) error {
	return m.lock.Release(MetadataKeys.MetadataClientLock(clientId, hash))
}

func (m *BlobCacheMetadata) StoreContentInBlobFs(ctx context.Context, path string, hash string, size uint64) error {
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
		err = m.SetFsNode(ctx, currentNodeId, metadata)
		if err != nil {
			return err
		}

		// Add the current node as a child of the previous node
		err = m.AddFsNodeChild(ctx, previousParentId, currentNodeId)
		if err != nil {
			return err
		}

		previousParentId = currentNodeId
	}

	return nil
}

func (m *BlobCacheMetadata) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	key := MetadataKeys.MetadataFsNode(id)

	res, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, &ErrNodeNotFound{Id: id}
	}

	metadata := &BlobFsMetadata{}
	if err = ToStruct(res, metadata); err != nil {
		return nil, fmt.Errorf("failed to deserialize blobfs node metadata <%v>: %v", key, err)
	}

	return metadata, nil
}

func (m *BlobCacheMetadata) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	key := MetadataKeys.MetadataFsNode(id)

	// If metadata exists, increment inode generation #
	res, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if len(res) > 0 {
		existingMetadata := &BlobFsMetadata{}
		if err = ToStruct(res, existingMetadata); err != nil {
			return err
		}

		metadata.Gen = existingMetadata.Gen + 1
	}

	err = m.rdb.HSet(ctx, key, ToSlice(metadata)).Err()
	if err != nil {
		return fmt.Errorf("failed to set blobfs node metadata <%v>: %w", key, err)
	}

	return nil
}

func (m *BlobCacheMetadata) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	nodeIds, err := m.rdb.SMembers(ctx, MetadataKeys.MetadataFsNodeChildren(id)).Result()
	if err != nil {
		return nil, err
	}

	entries := []*BlobFsMetadata{}
	for _, nodeId := range nodeIds {
		node, err := m.GetFsNode(ctx, nodeId)
		if err != nil {
			continue
		}

		entries = append(entries, node)
	}

	return entries, nil
}

func (m *BlobCacheMetadata) GetAvailableHosts(ctx context.Context, locality string, removeHostCallback func(host *BlobCacheHost)) ([]*BlobCacheHost, error) {
	hostAddrs, err := m.rdb.SMembers(ctx, MetadataKeys.MetadataHostIndex(locality)).Result()
	if err != nil {
		return nil, err
	}

	hosts := []*BlobCacheHost{}
	for _, addr := range hostAddrs {
		hostBytes, err := m.rdb.Get(ctx, MetadataKeys.MetadataHostKeepAlive(locality, addr)).Bytes()
		if err != nil {

			// If the keepalive key doesn't exist, remove the host index key
			if err == redis.Nil {
				m.RemoveHostFromIndex(ctx, locality, &BlobCacheHost{Addr: addr})
				removeHostCallback(&BlobCacheHost{Addr: addr})
			}

			continue
		}

		host := &BlobCacheHost{}
		if err = json.Unmarshal(hostBytes, host); err != nil {
			continue
		}

		hosts = append(hosts, host)
	}

	if len(hosts) == 0 {
		return nil, errors.New("no available hosts")
	}

	return hosts, nil
}

func (m *BlobCacheMetadata) AddHostToIndex(ctx context.Context, locality string, host *BlobCacheHost) error {
	return m.rdb.SAdd(ctx, MetadataKeys.MetadataHostIndex(locality), host.Addr).Err()
}

func (m *BlobCacheMetadata) RemoveHostFromIndex(ctx context.Context, locality string, host *BlobCacheHost) error {
	return m.rdb.SRem(ctx, MetadataKeys.MetadataHostIndex(locality), host.Addr).Err()
}

func (m *BlobCacheMetadata) SetHostKeepAlive(ctx context.Context, locality string, host *BlobCacheHost) error {
	hostBytes, err := json.Marshal(host)
	if err != nil {
		return err
	}

	return m.rdb.Set(ctx, MetadataKeys.MetadataHostKeepAlive(locality, host.Addr), hostBytes, time.Duration(defaultHostKeepAliveTimeoutS)*time.Second).Err()
}

func (m *BlobCacheMetadata) GetHostIndex(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	hostAddrs, err := m.rdb.SMembers(ctx, MetadataKeys.MetadataHostIndex(locality)).Result()
	if err != nil {
		return nil, err
	}

	hosts := []*BlobCacheHost{}
	for _, addr := range hostAddrs {
		hosts = append(hosts, &BlobCacheHost{Addr: addr})
	}

	return hosts, nil
}

func (m *BlobCacheMetadata) AddFsNodeChild(ctx context.Context, pid, id string) error {
	err := m.rdb.SAdd(ctx, MetadataKeys.MetadataFsNodeChildren(pid), id).Err()
	if err != nil {
		return err
	}
	return nil
}

func (m *BlobCacheMetadata) RemoveFsNodeChild(ctx context.Context, id string) error {
	return nil
}

func (m *BlobCacheMetadata) SetStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return m.lock.Acquire(ctx, MetadataKeys.MetadataStoreFromContentLock(sourcePath), RedisLockOptions{TtlS: storeFromContentLockTtlS, Retries: 0})
}

func (m *BlobCacheMetadata) RefreshStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return m.lock.Refresh(MetadataKeys.MetadataStoreFromContentLock(sourcePath), RedisLockOptions{TtlS: storeFromContentLockTtlS, Retries: 0})
}

func (m *BlobCacheMetadata) RemoveStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return m.lock.Release(MetadataKeys.MetadataStoreFromContentLock(sourcePath))
}

// Metadata key storage format
var (
	metadataPrefix               string = "blobcache"
	metadataHostIndex            string = "blobcache:host_index:%s"
	metadataClientLock           string = "blobcache:client_lock:%s:%s"
	metadataLocation             string = "blobcache:location:%s"
	metadataFsNode               string = "blobcache:fs:node:%s"
	metadataFsNodeChildren       string = "blobcache:fs:node:%s:children"
	metadataHostKeepAlive        string = "blobcache:host:keepalive:%s"
	metadataStoreFromContentLock string = "blobcache:store_from_content_lock:%s"
)

// Metadata keys
func (k *metadataKeys) MetadataPrefix() string {
	return metadataPrefix
}

func (k *metadataKeys) MetadataHostIndex(locality string) string {
	return fmt.Sprintf(metadataHostIndex, locality)
}

func (k *metadataKeys) MetadataHostKeepAlive(locality, addr string) string {
	return fmt.Sprintf(metadataHostKeepAlive, locality, addr)
}

func (k *metadataKeys) MetadataLocation(hash string) string {
	return fmt.Sprintf(metadataLocation, hash)
}

func (k *metadataKeys) MetadataFsNode(id string) string {
	return fmt.Sprintf(metadataFsNode, id)
}

func (k *metadataKeys) MetadataFsNodeChildren(id string) string {
	return fmt.Sprintf(metadataFsNodeChildren, id)
}

func (k *metadataKeys) MetadataClientLock(hostId, hash string) string {
	return fmt.Sprintf(metadataClientLock, hostId, hash)
}

func (k *metadataKeys) MetadataStoreFromContentLock(sourcePath string) string {
	sourcePath = strings.ReplaceAll(sourcePath, "/", "_")
	return fmt.Sprintf(metadataStoreFromContentLock, sourcePath)
}

var MetadataKeys = &metadataKeys{}

type metadataKeys struct{}
