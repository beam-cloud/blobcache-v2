package blobcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/dgraph-io/ristretto"
)

type ContentAddressableStorage struct {
	ctx         context.Context
	currentHost *BlobCacheHost
	cache       *ristretto.Cache
	config      BlobCacheConfig
	mu          sync.RWMutex
	metadata    *BlobCacheMetadata
}

func NewContentAddressableStorage(ctx context.Context, currentHost *BlobCacheHost, metadata *BlobCacheMetadata, config BlobCacheConfig) (*ContentAddressableStorage, error) {
	if config.MaxCacheSizeMb <= 0 || config.PageSizeBytes <= 0 {
		return nil, errors.New("invalid cache configuration")
	}

	cas := &ContentAddressableStorage{
		ctx:         ctx,
		config:      config,
		metadata:    metadata,
		currentHost: currentHost,
	}

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     config.MaxCacheSizeMb * 1e6,
		BufferItems: 64,
		OnEvict:     cas.onEvict,
	})
	if err != nil {
		return nil, err
	}

	cas.cache = cache
	return cas, nil
}

func (cas *ContentAddressableStorage) Add(ctx context.Context, content []byte, source string) (string, error) {
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	cas.mu.Lock()
	defer cas.mu.Unlock()

	size := int64(len(content))

	// Break content into chunks and store
	for offset := int64(0); offset < size; offset += cas.config.PageSizeBytes {
		chunkIdx := offset / cas.config.PageSizeBytes
		end := offset + cas.config.PageSizeBytes
		if end > size {
			end = size
		}

		chunk := content[offset:end]
		chunkKey := fmt.Sprintf("%s-%d", hashStr, chunkIdx)

		added := cas.cache.Set(chunkKey, chunk, int64(len(chunk)))
		if !added {
			return "", errors.New("unable to cache: set dropped")
		}
	}

	// Store entry
	err := cas.metadata.AddEntry(ctx, &BlobCacheEntry{
		Hash:    hashStr,
		Size:    size,
		Content: nil,
		Source:  source,
	}, cas.currentHost)
	if err != nil {
		return "", err
	}

	return hashStr, nil
}

func (cas *ContentAddressableStorage) Get(hash string, offset, length int64) ([]byte, error) {
	cas.mu.RLock()
	defer cas.mu.RUnlock()

	result := make([]byte, 0, length)
	remainingLength := length
	o := offset

	for remainingLength > 0 {
		chunkIdx := o / cas.config.PageSizeBytes
		chunkKey := fmt.Sprintf("%s-%d", hash, chunkIdx)

		// Check cache for chunk
		chunk, found := cas.cache.Get(chunkKey)
		if !found {
			return nil, errors.New("content not found")
		}

		chunkBytes := chunk.([]byte)
		start := o % cas.config.PageSizeBytes
		chunkRemaining := int64(len(chunkBytes)) - start
		if chunkRemaining <= 0 {
			break
		}

		readLength := min(remainingLength, chunkRemaining)
		end := start + readLength

		if start < 0 || end <= start || end > int64(len(chunkBytes)) {
			return nil, fmt.Errorf("invalid chunk boundaries: start %d, end %d, chunk size %d", start, end, len(chunkBytes))
		}

		result = append(result, chunkBytes[start:end]...)
		remainingLength -= readLength
		o += readLength
	}

	return result, nil
}

func (cas *ContentAddressableStorage) onEvict(item *ristretto.Item) {
	// logger.Info("evicted item: ", item.Key)

	cas.metadata.RemoveEntryLocation(cas.ctx, "mock", cas.currentHost)
	// TODO: make sure on eviction, we evict all chunks,
	// and update metadata to say we no longer have the file
}

func (cas *ContentAddressableStorage) Cleanup() {
	cas.cache.Close()
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
