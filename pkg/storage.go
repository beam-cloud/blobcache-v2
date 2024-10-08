package blobcache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beam-cloud/ristretto"
)

type ContentAddressableStorage struct {
	ctx         context.Context
	currentHost *BlobCacheHost
	cache       *ristretto.Cache[string, interface{}]
	config      BlobCacheConfig
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

	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,
		MaxCost:     config.MaxCacheSizeMb * 1e6,
		BufferItems: 64,
		OnEvict:     cas.onEvict,
		Metrics:     false,
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

func (cas *ContentAddressableStorage) Add(ctx context.Context, hash string, content []byte, sourcePath string, sourceOffset int64) error {
	size := int64(len(content))
	chunkKeys := []string{}

	// Break content into chunks and store
	for offset := int64(0); offset < size; offset += cas.config.PageSizeBytes {
		chunkIdx := offset / cas.config.PageSizeBytes
		end := offset + cas.config.PageSizeBytes
		if end > size {
			end = size
		}

		// Copy the chunk into a new buffer
		chunk := make([]byte, end-offset)
		copy(chunk, content[offset:end])
		chunkKey := fmt.Sprintf("%s-%d", hash, chunkIdx)

		_, exists := cas.cache.GetTTL(chunkKey)
		if exists {
			continue
		}

		// Store the chunk
		added := cas.cache.Set(chunkKey, cacheValue{Hash: hash, Content: chunk}, int64(len(chunk)))
		if !added {
			return errors.New("unable to cache: set dropped")
		}

		chunkKeys = append(chunkKeys, chunkKey)
	}

	// Release the large initial buffer
	content = nil

	// Store chunk keys in cache
	chunks := strings.Join(chunkKeys, ",")
	added := cas.cache.SetWithTTL(hash, chunks, int64(len(chunks)), time.Duration(cas.config.ObjectTtlS)*time.Second)
	if !added {
		return errors.New("unable to cache: set dropped")
	}

	// Store entry
	err := cas.metadata.AddEntry(ctx, &BlobCacheEntry{
		Hash:         hash,
		Size:         size,
		SourcePath:   sourcePath,
		SourceOffset: sourceOffset,
	}, cas.currentHost)
	if err != nil {
		return err
	}

	return nil
}

func (cas *ContentAddressableStorage) Get(hash string, offset, length int64) ([]byte, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, length))

	remainingLength := length
	o := offset

	cas.cache.ResetTTL(hash, time.Duration(cas.config.ObjectTtlS)*time.Second)

	for remainingLength > 0 {
		chunkIdx := o / cas.config.PageSizeBytes
		chunkKey := fmt.Sprintf("%s-%d", hash, chunkIdx)

		// Check cache for chunk
		value, found := cas.cache.Get(chunkKey)
		if !found {
			return nil, ErrContentNotFound
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

		if _, err := buffer.Write(chunkBytes[start:end]); err != nil {
			return nil, fmt.Errorf("failed to write to buffer: %v", err)
		}

		remainingLength -= readLength
		o += readLength
	}

	return buffer.Bytes(), nil
}

func (cas *ContentAddressableStorage) onEvict(item *ristretto.Item[interface{}]) {
	hash := ""
	var chunkKeys []string = []string{}

	// We've evicted a chunk of a cached object - extract the hash and evict all the other chunks
	switch v := item.Value.(type) {
	case cacheValue:
		hash = v.Hash
		chunks, found := cas.cache.Get(hash)
		if found {
			chunkKeys = strings.Split(chunks.(string), ",")
		}
	case string:
		// In this case, we evicted the key that stores which chunks are currently present in the cache
		// the value of which is formatted like this: "<hash>-0,<hash>-1,<hash>-2"
		// so here we can extract the hash by splitting on '-' and taking the first item
		hash = strings.SplitN(v, "-", 2)[0]
		chunkKeys = strings.Split(v, ",")
	default:
	}

	Logger.Infof("Evicted object: %s", hash)
	Logger.Debugf("Object chunks: %+v", chunkKeys)

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
