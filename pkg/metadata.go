package blobcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const (
	storeFromContentLockTtlS = 5
)

type BlobCacheMetadata struct {
	rdb  *RedisClient
	lock *RedisLock
}

func NewBlobCacheMetadata(cfg MetadataConfig) (*BlobCacheMetadata, error) {
	redisMode := RedisModeSingle
	if cfg.RedisMode != "" {
		redisMode = cfg.RedisMode
	}

	rdb, err := NewRedisClient(RedisConfig{
		Addrs:              []string{cfg.RedisAddr},
		Mode:               redisMode,
		Password:           cfg.RedisPasswd,
		SentinelPassword:   cfg.RedisPasswd,
		EnableTLS:          cfg.RedisTLSEnabled,
		MasterName:         cfg.RedisMasterName,
		InsecureSkipVerify: true, // HOTFIX: tailscale certs don't match in-cluster certs
	})
	if err != nil {
		return nil, err
	}

	lock := NewRedisLock(rdb)
	return &BlobCacheMetadata{
		rdb:  rdb,
		lock: lock,
	}, nil
}

func (m *BlobCacheMetadata) SetClientLock(ctx context.Context, clientId, hash string) error {
	err := m.lock.Acquire(ctx, MetadataKeys.MetadataClientLock(clientId, hash), RedisLockOptions{TtlS: 300, Retries: 0})
	if err != nil {
		return err
	}

	return nil
}

func (m *BlobCacheMetadata) RemoveClientLock(ctx context.Context, clientId, hash string) error {
	return m.lock.Release(MetadataKeys.MetadataClientLock(clientId, hash))
}

func (m *BlobCacheMetadata) GetFsNode(ctx context.Context, id string) (*BlobFsMetadata, error) {
	key := MetadataKeys.MetadataFsNode(id)

	res, err := m.rdb.HGetAll(ctx, key).Result()
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

	// If metadata exists, increment inode generation #
	res, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if len(res) > 0 {
		existingMetadata := &BlobFsMetadata{}
		if err = ToStruct(res, existingMetadata); err != nil {
			return err
		}

		metadata.Gen = existingMetadata.Gen + 1
	}

	err = m.rdb.HSet(ctx, key, ToSlice(metadata)).Err()
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

func (m *BlobCacheMetadata) GetAvailableHosts(ctx context.Context, locality string, removeHostCallback func(host *BlobCacheHost)) ([]*BlobCacheHost, error) {
	hostIds, err := m.rdb.SMembers(ctx, MetadataKeys.MetadataHostIndex(locality)).Result()
	if err != nil {
		return nil, err
	}

	hosts := []*BlobCacheHost{}
	for _, hostId := range hostIds {
		hostBytes, err := m.rdb.Get(ctx, MetadataKeys.MetadataHostKeepAlive(locality, hostId)).Bytes()
		if err != nil {

			// If the keepalive key doesn't exist, remove the host index key
			if err == redis.Nil {
				m.RemoveHostFromIndex(ctx, locality, &BlobCacheHost{HostId: hostId})
				removeHostCallback(&BlobCacheHost{HostId: hostId})
			}

			continue
		}

		host := &BlobCacheHost{}
		if err = json.Unmarshal(hostBytes, host); err != nil {
			continue
		}

		hosts = append(hosts, host)
	}

	if len(hosts) == 0 {
		return nil, errors.New("no available hosts")
	}

	return hosts, nil
}

func (m *BlobCacheMetadata) AddHostToIndex(ctx context.Context, locality string, host *BlobCacheHost) error {
	return m.rdb.SAdd(ctx, MetadataKeys.MetadataHostIndex(locality), host.HostId).Err()
}

func (m *BlobCacheMetadata) RemoveHostFromIndex(ctx context.Context, locality string, host *BlobCacheHost) error {
	return m.rdb.SRem(ctx, MetadataKeys.MetadataHostIndex(locality), host.HostId).Err()
}

func (m *BlobCacheMetadata) SetHostKeepAlive(ctx context.Context, locality string, host *BlobCacheHost) error {
	hostBytes, err := json.Marshal(host)
	if err != nil {
		return err
	}

	return m.rdb.Set(ctx, MetadataKeys.MetadataHostKeepAlive(locality, host.HostId), hostBytes, time.Duration(defaultHostKeepAliveTimeoutS)*time.Second).Err()
}

func (m *BlobCacheMetadata) GetHostIndex(ctx context.Context, locality string) ([]*BlobCacheHost, error) {
	hostIds, err := m.rdb.SMembers(ctx, MetadataKeys.MetadataHostIndex(locality)).Result()
	if err != nil {
		return nil, err
	}

	hosts := []*BlobCacheHost{}
	for _, hostId := range hostIds {
		hosts = append(hosts, &BlobCacheHost{HostId: hostId})
	}

	return hosts, nil
}

func (m *BlobCacheMetadata) AddFsNodeChild(ctx context.Context, pid, id string) error {
	err := m.rdb.SAdd(ctx, MetadataKeys.MetadataFsNodeChildren(pid), id).Err()
	if err != nil {
		return err
	}
	return nil
}

func (m *BlobCacheMetadata) RemoveFsNode(ctx context.Context, id string) error {
	return m.rdb.Del(ctx, MetadataKeys.MetadataFsNode(id)).Err()
}

func (m *BlobCacheMetadata) RemoveFsNodeChild(ctx context.Context, pid, id string) error {
	return m.rdb.SRem(ctx, MetadataKeys.MetadataFsNodeChildren(pid), id).Err()
}

func (m *BlobCacheMetadata) SetStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	return m.lock.Acquire(ctx, MetadataKeys.MetadataStoreFromContentLock(locality, sourcePath), RedisLockOptions{TtlS: storeFromContentLockTtlS, Retries: 0})
}

func (m *BlobCacheMetadata) RefreshStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	return m.lock.Refresh(MetadataKeys.MetadataStoreFromContentLock(locality, sourcePath), RedisLockOptions{TtlS: storeFromContentLockTtlS, Retries: 0})
}

func (m *BlobCacheMetadata) RemoveStoreFromContentLock(ctx context.Context, locality string, sourcePath string) error {
	return m.lock.Release(MetadataKeys.MetadataStoreFromContentLock(locality, sourcePath))
}

// Metadata key storage format
var (
	metadataPrefix               string = "blobcache"
	metadataHostIndex            string = "blobcache:host_index:%s"
	metadataClientLock           string = "blobcache:client_lock:%s:%s"
	metadataLocation             string = "blobcache:location:%s"
	metadataFsNode               string = "blobcache:fs:node:%s"
	metadataFsNodeChildren       string = "blobcache:fs:node:%s:children"
	metadataHostKeepAlive        string = "blobcache:host:keepalive:%s"
	metadataStoreFromContentLock string = "blobcache:store_from_content_lock:%s:%s"
)

// Metadata keys
func (k *metadataKeys) MetadataPrefix() string {
	return metadataPrefix
}

func (k *metadataKeys) MetadataHostIndex(locality string) string {
	return fmt.Sprintf(metadataHostIndex, locality)
}

func (k *metadataKeys) MetadataHostKeepAlive(locality, hostId string) string {
	return fmt.Sprintf(metadataHostKeepAlive, locality, hostId)
}

func (k *metadataKeys) MetadataLocation(hash string) string {
	return fmt.Sprintf(metadataLocation, hash)
}

func (k *metadataKeys) MetadataFsNode(id string) string {
	return fmt.Sprintf(metadataFsNode, id)
}

func (k *metadataKeys) MetadataFsNodeChildren(id string) string {
	return fmt.Sprintf(metadataFsNodeChildren, id)
}

func (k *metadataKeys) MetadataClientLock(hostId, hash string) string {
	return fmt.Sprintf(metadataClientLock, hostId, hash)
}

func (k *metadataKeys) MetadataStoreFromContentLock(locality, sourcePath string) string {
	sourcePath = strings.ReplaceAll(sourcePath, "/", "_")
	return fmt.Sprintf(metadataStoreFromContentLock, locality, sourcePath)
}

var MetadataKeys = &metadataKeys{}

type metadataKeys struct{}
