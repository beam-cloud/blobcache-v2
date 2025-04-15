package main

import (
	"context"
	"log"
	"os"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
)

func main() {
	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	cfg := configManager.GetConfig()

	blobcache.InitLogger(cfg.Global.DebugMode, cfg.Global.PrettyLogs)

	locality := os.Getenv("BLOBCACHE_LOCALITY")
	if locality == "" {
		blobcache.Logger.Infof("BLOBCACHE_LOCALITY is not set, using default locality: %s", cfg.Global.DefaultLocality)
		locality = cfg.Global.DefaultLocality
	}

	s, err := blobcache.NewCacheService(ctx, cfg, locality)
	if err != nil {
		log.Fatal(err)
	}

	s.StartServer(cfg.Global.ServerPort)
}
