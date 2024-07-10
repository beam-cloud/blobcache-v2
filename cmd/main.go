package main

import (
	"context"
	"log"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
)

func main() {
	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	ctx := context.Background()
	cfg := configManager.GetConfig()

	blobcache.InitLogger(cfg.DebugMode)
	defer blobcache.Logger.Sync()

	s, err := blobcache.NewCacheService(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	s.StartServer(cfg.Port)
}
