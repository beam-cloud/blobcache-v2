package blobcache

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

var cas *ContentAddressableStorage

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := cas.GetMetrics()

	//function outputs metrics in a plain text format that adheres
	//to the Prometheus exposition format (text/plain; version=0.0.4)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	fmt.Fprintf(w, "# HELP blobcache_uptime_seconds Uptime of blobcache in seconds\n")
	fmt.Fprintf(w, "blobcache_uptime_seconds %.2f\n", metrics.Uptime.Seconds())

	fmt.Fprintf(w, "# HELP blobcache_disk_usage_mb Disk cache usage in MB\n")
	fmt.Fprintf(w, "blobcache_disk_usage_mb %d\n", metrics.DiskUsageMb)

	fmt.Fprintf(w, "# HELP blobcache_disk_total_mb Total disk space in MB\n")
	fmt.Fprintf(w, "blobcache_disk_total_mb %d\n", metrics.TotalDiskSpaceMb)

	fmt.Fprintf(w, "# HELP blobcache_disk_usage_percentage Disk usage percentage\n")
	fmt.Fprintf(w, "blobcache_disk_usage_percentage %.2f\n", metrics.UsagePercentage)

	if metrics.RistrettoMetrics != nil {
		fmt.Fprintf(w, "# HELP blobcache_ram_hits RAM cache hits\n")
		fmt.Fprintf(w, "blobcache_ram_hits %d\n", metrics.RistrettoMetrics.Hits())

		fmt.Fprintf(w, "# HELP blobcache_ram_misses RAM cache misses\n")
		fmt.Fprintf(w, "blobcache_ram_misses %d\n", metrics.RistrettoMetrics.Misses())
	}
}

func StartMetricsServer() {
	ctx := context.Background()

	var err error
	cas, err = NewContentAddressableStorage(ctx, nil, "local", nil, BlobCacheConfig{
		Server: BlobCacheServerConfig{
			MaxCachePct:          50,
			PageSizeBytes:        4096,
			DiskCacheDir:         "/tmp/blobcache",
			DiskCacheMaxUsagePct: 80,
			ObjectTtlS:           600,
		},
		Global: BlobCacheGlobalConfig{
			DebugMode: true,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create ContentAddressableStorage: %v", err))
	}

	http.HandleFunc("/metrics", metricsHandler)

	// Friendly root handler to prevent 404 at /
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Blobcache metrics available at /metrics")
	})

	fmt.Println("Starting metrics server at :2112")
	if err := http.ListenAndServe(":2112", nil); err != nil {
		log.Fatalf("Metrics server failed to start: %v", err)
	}
}
