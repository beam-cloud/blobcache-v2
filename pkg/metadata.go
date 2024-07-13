package blobcache

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hanwen/go-fuse/v2/fuse"
	redis "github.com/redis/go-redis/v9"
)

type BlobCacheMetadata struct {
	rdb *RedisClient
}

const (
	redisMode RedisMode = "single"
)

func NewBlobCacheMetadata(cfg MetadataConfig) (*BlobCacheMetadata, error) {
	rdb, err := NewRedisClient(RedisConfig{
		Addrs:              []string{cfg.RedisAddr},
		Mode:               redisMode,
		Password:           cfg.RedisPasswd,
		EnableTLS:          cfg.RedisTLSEnabled,
		InsecureSkipVerify: true, // HOTFIX: tailscale certs don't match in-cluster certs
	})
	if err != nil {
		return nil, err
	}

	return &BlobCacheMetadata{
		rdb: rdb,
	}, nil
}

func (m *BlobCacheMetadata) AddEntry(ctx context.Context, entry *BlobCacheEntry, host *BlobCacheHost) error {
	entryKey := MetadataKeys.MetadataEntry(entry.Hash)

	exists, err := m.rdb.Exists(ctx, entryKey).Result()
	if err != nil {
		return err
	}

	// Entry not found, add it
	if exists == 0 {
		err := m.rdb.HSet(ctx, entryKey, ToSlice(entry)).Err()
		if err != nil {
			return fmt.Errorf("failed to set entry <%v>: %w", entryKey, err)
		}
	}

	// Add ref to entry
	return m.addEntryLocation(ctx, entry.Hash, host)
}

func (m *BlobCacheMetadata) RetrieveEntry(ctx context.Context, hash string) (*BlobCacheEntry, error) {
	entryKey := MetadataKeys.MetadataEntry(hash)

	res, err := m.rdb.HGetAll(context.TODO(), entryKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, &ErrEntryNotFound{Hash: hash}
	}

	entry := &BlobCacheEntry{}
	if err = ToStruct(res, entry); err != nil {
		return nil, fmt.Errorf("failed to deserialize entry <%v>: %v", entryKey, err)
	}

	return entry, nil
}

func (m *BlobCacheMetadata) RemoveEntryLocation(ctx context.Context, hash string, host *BlobCacheHost) error {
	err := m.rdb.SRem(ctx, MetadataKeys.MetadataLocation(hash), host.Addr).Err()
	if err != nil {
		return err
	}

	return m.rdb.Decr(ctx, MetadataKeys.MetadataRef(hash)).Err()
}

func (m *BlobCacheMetadata) GetEntryLocations(ctx context.Context, hash string) (mapset.Set[string], error) {
	hostAddrs, err := m.rdb.SMembers(ctx, MetadataKeys.MetadataLocation(hash)).Result()
	if err != nil {
		return nil, err
	}

	hostSet := mapset.NewSet[string]()
	for _, addr := range hostAddrs {
		hostSet.Add(addr)
	}

	return hostSet, nil
}

func (m *BlobCacheMetadata) addEntryLocation(ctx context.Context, hash string, host *BlobCacheHost) error {
	err := m.rdb.SAdd(ctx, MetadataKeys.MetadataLocation(hash), host.Addr).Err()
	if err != nil {
		return err
	}

	return m.rdb.Incr(ctx, MetadataKeys.MetadataRef(hash)).Err()
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

		// Since this is the last file, store as a file, not a dir
		if path == currentPath {
			metadata.Mode = fuse.S_IFREG | 0755
			metadata.Hash = hash
			metadata.Size = size
		}

		err = m.SetFsNode(ctx, currentNodeId, metadata)
		if err != nil {
			return err
		}

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

	res, err := m.rdb.HGetAll(context.TODO(), key).Result()
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

	err := m.rdb.HSet(ctx, key, ToSlice(metadata)).Err()
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

// Metadata key storage format
var (
	metadataPrefix         string = "blobcache"
	metadataEntry          string = "blobcache:entry:%s"
	metadataLocation       string = "blobcache:location:%s"
	metadataRef            string = "blobcache:ref:%s"
	metadataFsNode         string = "blobcache:fs:node:%s"
	metadataFsNodeChildren string = "blobcache:fs:node:%s:children"
)

// Metadata keys
func (k *metadataKeys) MetadataPrefix() string {
	return metadataPrefix
}

func (k *metadataKeys) MetadataEntry(hash string) string {
	return fmt.Sprintf(metadataEntry, hash)
}

func (k *metadataKeys) MetadataLocation(hash string) string {
	return fmt.Sprintf(metadataLocation, hash)
}

func (k *metadataKeys) MetadataRef(hash string) string {
	return fmt.Sprintf(metadataRef, hash)
}

func (k *metadataKeys) MetadataFsNode(id string) string {
	return fmt.Sprintf(metadataFsNode, id)
}

func (k *metadataKeys) MetadataFsNodeChildren(id string) string {
	return fmt.Sprintf(metadataFsNodeChildren, id)
}

var MetadataKeys = &metadataKeys{}

type metadataKeys struct{}
