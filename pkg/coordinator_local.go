package blobcache

import (
	"context"
)

type CoordinatorClient interface {
	AddHostToIndex(ctx context.Context, locality string, host *BlobCacheHost) error
	SetHostKeepAlive(ctx context.Context, locality string, host *BlobCacheHost) error
	GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error)
	GetRegionConfig(ctx context.Context, locality string) (BlobCacheServerConfig, error)
	SetClientLock(ctx context.Context, hash string, host string) error
	RemoveClientLock(ctx context.Context, hash string, host string) error
	SetStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error
	RemoveStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error
	RefreshStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error
	SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error
	GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error)
	GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error)
	AddFsNodeChild(ctx context.Context, pid, id string) error
}

type CoordinatorClientLocal struct {
	host         string
	globalConfig BlobCacheGlobalConfig
	serverConfig BlobCacheServerConfig
	metadata     *BlobCacheMetadata
}

func NewCoordinatorClientLocal(globalConfig BlobCacheGlobalConfig, serverConfig BlobCacheServerConfig) (CoordinatorClient, error) {
	metadata, err := NewBlobCacheMetadata(serverConfig.Metadata)
	if err != nil {
		return nil, err
	}

	return &CoordinatorClientLocal{globalConfig: globalConfig, serverConfig: serverConfig, metadata: metadata}, nil
}

func (c *CoordinatorClientLocal) GetRegionConfig(ctx context.Context, locality string) (BlobCacheServerConfig, error) {
	return c.serverConfig, nil
}

func (c *CoordinatorClientLocal) AddHostToIndex(ctx context.Context, locality string, host *BlobCacheHost) error {
	return c.metadata.AddHostToIndex(ctx, locality, host)
}

func (c *CoordinatorClientLocal) SetHostKeepAlive(ctx context.Context, locality string, host *BlobCacheHost) error {
	return c.metadata.SetHostKeepAlive(ctx, locality, host)
}

func (c *CoordinatorClientLocal) SetStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	return c.metadata.SetStoreFromContentLock(ctx, locality, sourcePath)
}

func (c *CoordinatorClientLocal) RemoveStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	return c.metadata.RemoveStoreFromContentLock(ctx, locality, sourcePath)
}

func (c *CoordinatorClientLocal) RefreshStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	return c.metadata.RefreshStoreFromContentLock(ctx, locality, sourcePath)
}

func (c *CoordinatorClientLocal) GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	return c.metadata.GetAvailableHosts(ctx, locality, func(host *BlobCacheHost) {
		c.metadata.RemoveHostFromIndex(ctx, locality, host)
	})
}

func (c *CoordinatorClientLocal) SetClientLock(ctx context.Context, hash string, host string) error {
	return c.metadata.SetClientLock(ctx, hash, host)
}

func (c *CoordinatorClientLocal) RemoveClientLock(ctx context.Context, hash string, host string) error {
	return c.metadata.RemoveClientLock(ctx, hash, host)
}

func (c *CoordinatorClientLocal) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	return c.metadata.SetFsNode(ctx, id, metadata)
}

func (c *CoordinatorClientLocal) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	return c.metadata.GetFsNode(ctx, id)
}

func (c *CoordinatorClientLocal) AddFsNodeChild(ctx context.Context, pid, id string) error {
	return c.metadata.AddFsNodeChild(ctx, pid, id)
}

func (c *CoordinatorClientLocal) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	return c.metadata.GetFsNodeChildren(ctx, id)
}
