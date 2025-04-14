package main

import (
	"context"
	"flag"
	"log"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
)

func main() {
	flag.Parse()

	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg := configManager.GetConfig()

	// Initialize logger
	blobcache.InitLogger(cfg.Global.DebugMode, cfg.Global.PrettyLogs)

	ctx := context.Background()

	client, err := blobcache.NewBlobCacheClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Unable to create client: %v", err)
	}

	err = client.GetState()
	if err != nil {
		log.Fatalf("Unable to get state: %v", err)
	}

}
