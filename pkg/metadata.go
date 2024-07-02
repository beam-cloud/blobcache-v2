package blobcache

import (
	"context"
	"fmt"

	mapset "github.com/deckarep/golang-set/v2"
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

func (m *BlobCacheMetadata) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	key := MetadataKeys.MetadataFsNode(id)

	res, err := m.rdb.HGetAll(context.TODO(), key).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, &ErrFileNotFound{Id: id}
	}

	metadata := &BlobFsMetadata{}
	if err = ToStruct(res, metadata); err != nil {
		return nil, fmt.Errorf("failed to deserialize blobfs metadata <%v>: %v", key, err)
	}

	return metadata, nil
}

func (m *BlobCacheMetadata) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	key := MetadataKeys.MetadataFsNode(id)

	exists, err := m.rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	}

	// Metadata not found, add it
	if exists == 0 {
		err := m.rdb.HSet(ctx, key, ToSlice(metadata)).Err()
		if err != nil {
			return fmt.Errorf("failed to set dir metadata <%v>: %w", key, err)
		}
	}

	return nil
}

func (m *BlobCacheMetadata) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	return nil, nil
}

func (m *BlobCacheMetadata) AddFsNodeChild(ctx context.Context, pid, id string) error {
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
