package blobcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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

type cacheValue struct {
	Hash    string
	Content []byte
}

func (cas *ContentAddressableStorage) Add(ctx context.Context, content []byte, source string) (string, error) {
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	size := int64(len(content))
	chunkKeys := []string{}

	start := time.Now()

	Logger.Debugf("Adding content")

	// Break content into chunks and store
	for offset := int64(0); offset < size; offset += cas.config.PageSizeBytes {
		chunkIdx := offset / cas.config.PageSizeBytes
		end := offset + cas.config.PageSizeBytes
		if end > size {
			end = size
		}

		chunk := content[offset:end]

		chunkKey := fmt.Sprintf("%s-%d", hashStr, chunkIdx)

		added := cas.cache.Set(chunkKey, cacheValue{Hash: hashStr, Content: chunk}, int64(len(chunk)))
		if !added {
			return "", errors.New("unable to cache: set dropped")
		}

		chunkKeys = append(chunkKeys, chunkKey)
	}

	Logger.Debugf("Finished adding content in %v", time.Since(start))
	chunks := strings.Join(chunkKeys, ",")
	added := cas.cache.Set(hashStr, chunks, int64(len(chunks)))
	if !added {
		return "", errors.New("unable to cache: set dropped")
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
	result := make([]byte, 0, length)
	remainingLength := length
	o := offset

	for remainingLength > 0 {
		chunkIdx := o / cas.config.PageSizeBytes
		chunkKey := fmt.Sprintf("%s-%d", hash, chunkIdx)

		// Check cache for chunk
		value, found := cas.cache.Get(chunkKey)
		if !found {
			return nil, errors.New("content not found")
		}

		v := value.(cacheValue)

		chunkBytes := v.Content
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
	hash := ""
	var chunkKeys []string = []string{}

	// We've evicted a chunk of a cached object - extract the hash and evict all the other chunks
	v, ok := item.Value.(cacheValue)
	if ok {
		hash = v.Hash
		chunks, found := cas.cache.Get(hash)
		if found {
			chunkKeys = strings.Split(chunks.(string), ",")
		}
	} else {
		// In this case, we evicted the key that stores which chunks are currently present in the cache
		// the value of which is formatted like this: "<hash>-0,<hash>-1,<hash>-2"
		// so here we can extract the hash by splitting on '-' and taking the first item
		v, ok := item.Value.(string)
		if ok {
			hash = strings.SplitN(v, "-", 2)[0]
		}

		chunkKeys = strings.Split(v, ",")
	}

	Logger.Debugf("Evicted object: %s", hash)

	for _, k := range chunkKeys {
		cas.cache.Del(k)
	}

	// Remove location of this cached content
	cas.metadata.RemoveEntryLocation(cas.ctx, hash, cas.currentHost)
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
