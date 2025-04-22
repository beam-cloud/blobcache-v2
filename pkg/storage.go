package blobcache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/beam-cloud/ristretto"
	"github.com/shirou/gopsutil/mem"
)

const (
	diskCacheUsageCheckInterval = 1 * time.Minute
)

type ContentAddressableStorage struct {
	ctx                     context.Context
	currentHost             *BlobCacheHost
	locality                string
	cache                   *ristretto.Cache[string, interface{}]
	serverConfig            BlobCacheServerConfig
	globalConfig            BlobCacheGlobalConfig
	coordinator             CoordinatorClient
	maxCacheSizeMb          int64
	diskCacheDir            string
	diskCachedUsageExceeded bool
	mu                      sync.Mutex
}

func NewContentAddressableStorage(ctx context.Context, currentHost *BlobCacheHost, locality string, coordinator CoordinatorClient, config BlobCacheConfig) (*ContentAddressableStorage, error) {
	if config.Server.MaxCachePct <= 0 || config.Server.PageSizeBytes <= 0 {
		return nil, errors.New("invalid cache configuration")
	}

	cas := &ContentAddressableStorage{
		ctx:          ctx,
		serverConfig: config.Server,
		globalConfig: config.Global,
		coordinator:  coordinator,
		currentHost:  currentHost,
		locality:     locality,
		diskCacheDir: config.Server.DiskCacheDir,
		mu:           sync.Mutex{},
	}

	Logger.Infof("Disk cache directory located at: '%s'", cas.diskCacheDir)

	availableMemoryMb := getAvailableMemoryMb()
	maxCacheSizeMb := (availableMemoryMb * cas.serverConfig.MaxCachePct) / 100
	maxCost := maxCacheSizeMb * 1e6

	Logger.Infof("Total available memory: %dMB", availableMemoryMb)
	Logger.Infof("Max cache size: %dMB", maxCacheSizeMb)
	Logger.Infof("Max cost: %d", maxCost)

	if maxCacheSizeMb <= 0 {
		return nil, errors.New("invalid memory limit")
	}

	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,
		MaxCost:     maxCost,
		BufferItems: 64,
		OnEvict:     cas.onEvict,
		Metrics:     cas.globalConfig.DebugMode,
	})
	if err != nil {
		return nil, err
	}

	cas.cache = cache
	cas.maxCacheSizeMb = maxCacheSizeMb

	go cas.monitorDiskCacheUsage()

	return cas, nil
}

func getAvailableMemoryMb() int64 {
	v, err := mem.VirtualMemory()
	if err != nil {
		log.Fatalf("Unable to retrieve host memory info: %v", err)
	}
	return int64(v.Total / (1024 * 1024))
}

type cacheValue struct {
	Hash    string
	Content []byte
}

func (cas *ContentAddressableStorage) Add(ctx context.Context, hash string, content []byte) error {
	size := int64(len(content))
	chunkKeys := []string{}

	if cas.globalConfig.DebugMode {
		Logger.Debugf("Cost added before Add: %+v", cas.cache.Metrics.CostAdded())
	}

	dirPath := filepath.Join(cas.diskCacheDir, hash)
	if !cas.diskCachedUsageExceeded {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	// Break content into chunks and store
	for offset := int64(0); offset < size; offset += cas.serverConfig.PageSizeBytes {
		chunkIdx := offset / cas.serverConfig.PageSizeBytes
		end := offset + cas.serverConfig.PageSizeBytes
		if end > size {
			end = size
		}

		// Copy the chunk into a new buffer
		chunk := make([]byte, end-offset)
		copy(chunk, content[offset:end])
		chunkKey := fmt.Sprintf("%s-%d", hash, chunkIdx)

		// Write through to disk cache if we still have storage available
		if !cas.diskCachedUsageExceeded {
			filePath := filepath.Join(dirPath, chunkKey)
			if err := os.WriteFile(filePath, chunk, 0644); err != nil {
				return fmt.Errorf("failed to write to disk cache: %w", err)
			}
		}

		chunkKeys = append(chunkKeys, chunkKey)

		_, exists := cas.cache.GetTTL(chunkKey)
		if exists {
			continue
		}

		// Store the chunk
		added := cas.cache.Set(chunkKey, cacheValue{Hash: hash, Content: chunk}, int64(len(chunk)))
		if !added {
			return errors.New("unable to cache: set dropped")
		}

	}

	// Release the large initial buffer
	content = nil

	// Store chunk keys in cache
	chunks := strings.Join(chunkKeys, ",")
	added := cas.cache.SetWithTTL(hash, chunks, int64(len(chunks)), time.Duration(cas.serverConfig.ObjectTtlS)*time.Second)
	if !added {
		return errors.New("unable to cache: set dropped")
	}

	Logger.Debugf("Added object: %s, size: %d bytes", hash, size)
	return nil
}

func (cas *ContentAddressableStorage) Exists(hash string) bool {
	var exists bool = false

	_, exists = cas.cache.GetTTL(hash)
	if !exists {
		exists, err := os.Stat(filepath.Join(cas.diskCacheDir, hash))
		if err != nil {
			return false
		}

		return exists.IsDir()
	}

	return exists
}

func (cas *ContentAddressableStorage) Get(hash string, offset, length int64, dst []byte) (int64, error) {
	remainingLength := length
	o := offset
	dstOffset := int64(0)

	cas.cache.ResetTTL(hash, time.Duration(cas.serverConfig.ObjectTtlS)*time.Second)

	for remainingLength > 0 {
		chunkIdx := o / cas.serverConfig.PageSizeBytes
		chunkKey := fmt.Sprintf("%s-%d", hash, chunkIdx)

		var value interface{}
		var found bool = false

		// Check in-memory cache for chunk
		value, found = cas.cache.Get(chunkKey)

		// Not found in memory, check disk cache before giving up
		if !found {
			var err error
			value, found, err = cas.getFromDiskCache(hash, chunkKey)
			if err != nil || !found {
				return 0, err
			}
		}

		v, ok := value.(cacheValue)
		if !ok {
			return 0, fmt.Errorf("unexpected cache value type")
		}

		chunkBytes := v.Content
		start := o % cas.serverConfig.PageSizeBytes
		chunkRemaining := int64(len(chunkBytes)) - start
		if chunkRemaining <= 0 {
			break
		}

		readLength := min(remainingLength, chunkRemaining)
		end := start + readLength

		if start < 0 || end <= start || end > int64(len(chunkBytes)) {
			return 0, fmt.Errorf("invalid chunk boundaries: start %d, end %d, chunk size %d", start, end, len(chunkBytes))
		}

		copy(dst[dstOffset:dstOffset+readLength], chunkBytes[start:end])

		remainingLength -= readLength
		o += readLength
		dstOffset += readLength
	}

	return dstOffset, nil
}

func (cas *ContentAddressableStorage) getFromDiskCache(hash, chunkKey string) (value cacheValue, found bool, err error) {
	cas.mu.Lock()
	defer cas.mu.Unlock()

	rawValue, found := cas.cache.Get(chunkKey)
	if found {
		return rawValue.(cacheValue), true, nil
	}

	chunkPath := filepath.Join(cas.diskCacheDir, hash, chunkKey)
	chunkBytes, err := os.ReadFile(chunkPath)
	if err != nil {
		return cacheValue{}, false, ErrContentNotFound
	}

	value = cacheValue{Hash: hash, Content: chunkBytes}
	cas.cache.Set(chunkKey, value, int64(len(chunkBytes)))

	return value, true, nil
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

func (cas *ContentAddressableStorage) monitorDiskCacheUsage() {
	ticker := time.NewTicker(diskCacheUsageCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cas.ctx.Done():
			return
		case <-ticker.C:
			usage, err := getDiskUsageMb(cas.diskCacheDir)
			if err == nil {
				totalDiskSpace, err := getTotalDiskSpaceMb(cas.diskCacheDir)
				if err == nil {
					usagePct := float64(usage) / float64(totalDiskSpace)

					Logger.Infof("Disk cache usage: %dMB / %dMB (%.2f%%)", usage, totalDiskSpace, usagePct*100)

					cas.mu.Lock()
					cas.diskCachedUsageExceeded = usagePct > cas.serverConfig.DiskCacheMaxUsagePct
					cas.mu.Unlock()
				}
			}
		}
	}
}

func getDiskUsageMb(path string) (int64, error) {
	var totalUsage int64 = 0
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalUsage += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return totalUsage / (1024 * 1024), nil
}

func getTotalDiskSpaceMb(path string) (int64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return int64(stat.Blocks) * int64(stat.Bsize) / (1024 * 1024), nil
}
