package blobcache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/dgraph-io/ristretto"
)

type ContentAddressableStorage struct {
	inMemory *ristretto.Cache
	config   BlobCacheConfig
	mu       sync.RWMutex
}

func NewContentAddressableStorage(config BlobCacheConfig) (*ContentAddressableStorage, error) {
	if config.MaxCacheSizeMb <= 0 || config.PageSizeBytes <= 0 {
		return nil, errors.New("invalid cache configuration")
	}

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     config.MaxCacheSizeMb * 1e6,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}

	return &ContentAddressableStorage{
		inMemory: cache,
	}, nil
}

func (cas *ContentAddressableStorage) Add(content []byte) (string, error) {
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	cas.mu.Lock()
	defer cas.mu.Unlock()

	// Break content into chunks and store on disk
	for offset := int64(0); offset < int64(len(content)); offset += cas.config.PageSizeBytes {
		chunkIdx := offset / cas.config.PageSizeBytes
		end := offset + cas.config.PageSizeBytes
		if end > int64(len(content)) {
			end = int64(len(content))
		}

		chunk := content[offset:end]
		chunkKey := fmt.Sprintf("%s-%d", hashStr, chunkIdx)
		cas.inMemory.Set(chunkKey, chunk, int64(len(chunk)))
	}

	return hashStr, nil
}

func (cas *ContentAddressableStorage) Get(hash string, offset, length int64) ([]byte, error) {
	cas.mu.RLock()
	defer cas.mu.RUnlock()

	var result []byte
	remainingLength := length

	o := offset
	for remainingLength > 0 {
		var chunkBytes []byte
		chunkIdx := o / cas.config.PageSizeBytes
		chunkKey := fmt.Sprintf("%s-%d", hash, chunkIdx)

		// Check in-memory cache first
		if chunk, found := cas.inMemory.Get(chunkKey); found {
			chunkBytes = chunk.([]byte)
		}

		start := o % cas.config.PageSizeBytes
		chunkRemaining := int64(len(chunkBytes)) - start
		if chunkRemaining <= 0 {
			// No more data in this chunk, break out of the loop
			break
		}

		readLength := min(remainingLength, chunkRemaining)
		end := start + readLength

		// Validate start and end positions
		if start < 0 || end <= start || end > int64(len(chunkBytes)) {
			return nil, fmt.Errorf("invalid chunk boundaries: start %d, end %d, chunk size %d", start, end, len(chunkBytes))
		}

		// Read only the required portion of the chunk
		requiredChunkBytes := chunkBytes[start:end]

		// Append the required bytes to the result
		result = append(result, requiredChunkBytes...)

		// Update the remaining length and current offset for the next iteration
		remainingLength -= readLength
		o += readLength
	}

	return result, nil
}

func (cas *ContentAddressableStorage) Cleanup() {
	cas.inMemory.Close()
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
