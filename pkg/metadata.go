package blobcache

import (
	"context"
	"fmt"

	"github.com/beam-cloud/beta9/pkg/common"
	"github.com/beam-cloud/beta9/pkg/types"
)

type BlobCacheMetadata struct {
	rdb *common.RedisClient
}

func NewBlobCacheMetadata(cfg MetadataConfig) (*BlobCacheMetadata, error) {
	rdb, err := common.NewRedisClient(types.RedisConfig{
		Addrs:              []string{cfg.RedisAddr},
		Mode:               "single",
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

	// TODO: first check if entry exists in redis setting anything
	err := m.rdb.HSet(context.TODO(), entryKey, common.ToSlice(entry)).Err()

	if err != nil {
		return fmt.Errorf("failed to set entry <%v>: %w", entryKey, err)
	}

	return m.addEntryLocation(ctx, host)
}

func (m *BlobCacheMetadata) RetrieveEntry(ctx context.Context) error {
	return m.rdb.Set(ctx, MetadataKeys.MetadataPrefix(), "true", 0).Err()
}

func (m *BlobCacheMetadata) addEntryLocation(ctx context.Context, host *BlobCacheHost) error {
	return m.rdb.Set(ctx, MetadataKeys.MetadataPrefix(), "true", 0).Err()
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
