package main

import (
	"log"

	blobcache "github.com/beam-cloud/blobcache/pkg"
)

func main() {
	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	cfg := configManager.GetConfig()
	client, err := blobcache.NewBlobCacheClient(cfg)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	log.Println("CONFIG: ", cfg)

	host, err := client.GetNearestHost()
	if err != nil {
		log.Printf("err finding host: %v\n", err)
	}

	log.Printf("Found host: %+v\n", host)
}
