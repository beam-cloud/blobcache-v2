package blobcache

import (
	"context"
)

type CoordinatorClient interface {
	AddHostToIndex(ctx context.Context, host *BlobCacheHost) error
	SetHostKeepAlive(ctx context.Context, host *BlobCacheHost) error
	GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error)
	SetClientLock(ctx context.Context, hash string, host string) error
	RemoveClientLock(ctx context.Context, hash string, host string) error
	SetStoreFromContentLock(ctx context.Context, sourcePath string) error
	RemoveStoreFromContentLock(ctx context.Context, sourcePath string) error
	RefreshStoreFromContentLock(ctx context.Context, sourcePath string) error
	SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error
	GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error)
	GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error)
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

func (c *CoordinatorClientLocal) AddHostToIndex(ctx context.Context, host *BlobCacheHost) error {
	return c.metadata.AddHostToIndex(ctx, host)
}

func (c *CoordinatorClientLocal) SetHostKeepAlive(ctx context.Context, host *BlobCacheHost) error {
	return c.metadata.SetHostKeepAlive(ctx, host)
}

func (c *CoordinatorClientLocal) SetStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return c.metadata.SetStoreFromContentLock(ctx, sourcePath)
}

func (c *CoordinatorClientLocal) RemoveStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return c.metadata.RemoveStoreFromContentLock(ctx, sourcePath)
}

func (c *CoordinatorClientLocal) RefreshStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return c.metadata.RefreshStoreFromContentLock(ctx, sourcePath)
}

func (c *CoordinatorClientLocal) GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	hosts := make([]*BlobCacheHost, 0)

	return hosts, nil
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

func (c *CoordinatorClientLocal) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	return c.metadata.GetFsNodeChildren(ctx, id)
}
