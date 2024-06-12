package blobcache

import (
	"context"

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
		EnableTLS:          false,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}

	return &BlobCacheMetadata{
		rdb: rdb,
	}, nil
}

func (m *BlobCacheMetadata) AddEntry(ctx context.Context) error {
	return nil
}
