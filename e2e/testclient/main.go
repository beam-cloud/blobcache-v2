package main

import (
	"context"
	"log"
	"time"

	blobcache "github.com/beam-cloud/blobcache/pkg"
)

func main() {
	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	cfg := configManager.GetConfig()
	ctx := context.Background()

	client, err := blobcache.NewBlobCacheClient(ctx, cfg)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	for {
		client.GetState()
		time.Sleep(time.Second)
	}

}
