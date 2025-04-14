package blobcache

import (
	"context"

	mapset "github.com/deckarep/golang-set/v2"
)

type MetadataClientCoordinator struct {
	cfg  BlobCacheGlobalConfig
	host string
}

func NewMetadataClientCoordinator(cfg BlobCacheGlobalConfig, token string) (MetadataClient, error) {
	return &MetadataClientCoordinator{cfg: cfg, host: cfg.CoordinatorHost}, nil
}

func (c *MetadataClientCoordinator) AddHostToIndex(ctx context.Context, host *BlobCacheHost) error {
	return nil
}

func (c *MetadataClientCoordinator) SetHostKeepAlive(ctx context.Context, host *BlobCacheHost) error {
	return nil
}

func (c *MetadataClientCoordinator) AddEntry(ctx context.Context, entry *BlobCacheEntry, host *BlobCacheHost) error {
	return nil
}

func (c *MetadataClientCoordinator) RemoveEntry(ctx context.Context, hash string) error {
	return nil
}

func (c *MetadataClientCoordinator) RemoveEntryLocation(ctx context.Context, hash string, host *BlobCacheHost) error {
	return nil
}

func (c *MetadataClientCoordinator) SetStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *MetadataClientCoordinator) RemoveStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *MetadataClientCoordinator) RefreshStoreFromContentLock(ctx context.Context, sourcePath string) error {
	return nil
}

func (c *MetadataClientCoordinator) GetAvailableHosts(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	hosts := make([]*BlobCacheHost, 0)

	return hosts, nil
}

func (c *MetadataClientCoordinator) GetEntryLocations(ctx context.Context, hash string) (mapset.Set[string], error) {
	hostAddrs := []string{}

	hostSet := mapset.NewSet[string]()
	for _, addr := range hostAddrs {
		hostSet.Add(addr)
	}

	return hostSet, nil
}

func (c *MetadataClientCoordinator) SetClientLock(ctx context.Context, hash string, host string) error {
	return nil
}

func (c *MetadataClientCoordinator) RemoveClientLock(ctx context.Context, hash string, host string) error {
	return nil
}

func (c *MetadataClientCoordinator) RetrieveEntry(ctx context.Context, hash string) (*BlobCacheEntry, error) {
	return nil, &ErrEntryNotFound{Hash: hash}
}

func (c *MetadataClientCoordinator) SetFsNode(ctx context.Context, id string, metadata *BlobFsMetadata) error {
	return nil
}

func (c *MetadataClientCoordinator) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	return nil, nil
}

func (c *MetadataClientCoordinator) GetFsNodeChildren(ctx context.Context, id string) ([]*BlobFsMetadata, error) {
	return nil, nil
}
