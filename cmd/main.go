package main

import (
	"log"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
)

func main() {
	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	config := configManager.GetConfig()

	s, err := blobcache.NewCacheService(config)
	if err != nil {
		log.Fatal(err)
	}

	s.StartServer(config.Port)
}
