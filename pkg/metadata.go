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

func (m *BlobCacheMetadata) AddEntry(ctx context.Context) error {
	return m.rdb.Set(ctx, MetadataKeys.MetadataPrefix(), "true", 0).Err()
}

// Metadata key storage format
var (
	metadataPrefix string = "blobcache"
	metadataEntry  string = "blobcache:entry:%s"
)

// Workspace keys
func (k *metadataKeys) MetadataPrefix() string {
	return metadataPrefix
}

func (k *metadataKeys) MetadataEntry(hash string) string {
	return fmt.Sprintf(metadataEntry, hash)
}

var MetadataKeys = &metadataKeys{}

type metadataKeys struct{}
