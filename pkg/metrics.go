package blobcache

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

type BlobcacheMetrics struct {
	DiskCacheUsageMB  *metrics.Histogram
	DiskCacheUsagePct *metrics.Histogram
	MemCacheUsageMB   *metrics.Histogram
	MemCacheUsagePct  *metrics.Histogram
}

func initMetrics(ctx context.Context, config BlobCacheMetricsConfig, currentHost *BlobCacheHost) BlobcacheMetrics {
	username := config.Username
	password := config.Password
	credentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	opts := &metrics.PushOptions{
		Headers: []string{
			fmt.Sprintf("Authorization: Basic %s", credentials),
		},
	}

	pushURL := config.URL
	interval := time.Duration(config.PushIntervalS) * time.Second
	pushProcessMetrics := true

	err := metrics.InitPushWithOptions(ctx, pushURL, interval, pushProcessMetrics, opts)
	if err != nil {
		Logger.Errorf("Failed to initialize metrics: %v", err)
	}

	diskCacheUsageMB := metrics.NewHistogram(`blobcache_disk_cache_usage_mb{host="` + currentHost.HostId + `"}`)
	diskCacheUsagePct := metrics.NewHistogram(`blobcache_disk_cache_usage_pct{host="` + currentHost.HostId + `"}`)
	memCacheUsageMB := metrics.NewHistogram(`blobcache_mem_cache_usage_mb{host="` + currentHost.HostId + `"}`)
	memCacheUsagePct := metrics.NewHistogram(`blobcache_mem_cache_usage_pct{host="` + currentHost.HostId + `"}`)

	return BlobcacheMetrics{
		DiskCacheUsageMB:  diskCacheUsageMB,
		DiskCacheUsagePct: diskCacheUsagePct,
		MemCacheUsageMB:   memCacheUsageMB,
		MemCacheUsagePct:  memCacheUsagePct,
	}
}
