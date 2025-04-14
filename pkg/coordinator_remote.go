package blobcache

import (
	"context"
)

type CoordinatorClientRemote struct {
	cfg  BlobCacheGlobalConfig
	host string
}

func NewCoordinatorClientRemote(cfg BlobCacheGlobalConfig, token string) (CoordinatorClient, error) {
	return &CoordinatorClientRemote{cfg: cfg, host: cfg.CoordinatorHost}, nil
}

func (c *CoordinatorClientRemote) AddHostToIndex(ctx context.Context, host *BlobCacheHost) error {
	return nil
}

func (c *CoordinatorClientRemote) SetHostKeepAlive(ctx context.Context, host *BlobCacheHost) error {
	return nil
}

func (c *CoordinatorClientRemote) AddEntry(ctx context.Context, entry *BlobCacheEntry, host *BlobCacheHost) error {
	return nil
}

func (c *CoordinatorClientRemote) RemoveEntry(ctx context.Context, hash string) error {
	return nil
}

func (c *CoordinatorClientRemote) RemoveEntryLocation(ctx context.Context, hash string, host *BlobCacheHost) error {
	return nil
}

func (c *CoordinatorClientRemote) SetStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *CoordinatorClientRemote) RemoveStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *CoordinatorClientRemote) RefreshStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *CoordinatorClientRemote) GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	hosts := make([]*BlobCacheHost, 0)

	return hosts, nil
}

func (c *CoordinatorClientRemote) SetClientLock(ctx context.Context, hash string, host string) error {
	return nil
}

func (c *CoordinatorClientRemote) RemoveClientLock(ctx context.Context, hash string, host string) error {
	return nil
}

func (c *CoordinatorClientRemote) RetrieveEntry(ctx context.Context, hash string) (*BlobCacheEntry, error) {
	return nil, &ErrEntryNotFound{Hash: hash}
}

func (c *CoordinatorClientRemote) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	return nil
}

func (c *CoordinatorClientRemote) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	return nil, nil
}

func (c *CoordinatorClientRemote) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	return nil, nil
}
