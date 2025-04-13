package main

import (
	"context"
	"log"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
)

func main() {
	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	cfg := configManager.GetConfig()

	blobcache.InitLogger(cfg.Server.DebugMode, cfg.Server.PrettyLogs)

	s, err := blobcache.NewCacheService(ctx, cfg.Server)
	if err != nil {
		log.Fatal(err)
	}

	s.StartServer(cfg.Server.Port)
}
