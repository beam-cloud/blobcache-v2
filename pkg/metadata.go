package blobcache

import (
	"context"
	"fmt"

	"github.com/beam-cloud/beta9/pkg/common"
	"github.com/beam-cloud/beta9/pkg/types"
	mapset "github.com/deckarep/golang-set/v2"
	redis "github.com/redis/go-redis/v9"
)

type BlobCacheMetadata struct {
	rdb *common.RedisClient
}

const (
	redisMode types.RedisMode = "single"
)

func NewBlobCacheMetadata(cfg MetadataConfig) (*BlobCacheMetadata, error) {
	rdb, err := common.NewRedisClient(types.RedisConfig{
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

	// m.rdb.ZAdd()
	// m.rdb.ZRangeByScore()

	// Entry not found, add it
	if exists == 0 {
		err := m.rdb.HSet(ctx, entryKey, common.ToSlice(entry)).Err()
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
	if err = common.ToStruct(res, entry); err != nil {
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

	return hostSet, err
}

func (m *BlobCacheMetadata) addEntryLocation(ctx context.Context, hash string, host *BlobCacheHost) error {
	err := m.rdb.SAdd(ctx, MetadataKeys.MetadataLocation(hash), host.Addr).Err()
	if err != nil {
		return err
	}

	return m.rdb.Incr(ctx, MetadataKeys.MetadataRef(hash)).Err()
}

// Metadata key storage format
var (
	metadataPrefix   string = "blobcache"
	metadataEntry    string = "blobcache:entry:%s"
	metadataLocation string = "blobcache:location:%s"
	metadataRef      string = "blobcache:ref:%s"
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

var MetadataKeys = &metadataKeys{}

type metadataKeys struct{}
