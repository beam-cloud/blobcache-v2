package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"log"
	"os"
	"time"

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

	err = client.WaitForHosts(time.Second * 10)
	if err != nil {
		log.Fatalf("Unable to wait for hosts: %v", err)
	}

	content := "Hello, World!"
	hashBytes := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(hashBytes[:])

	gotContent, err := client.GetContent(hash, 0, int64(len(content)), struct {
		RoutingKey string
	}{
		RoutingKey: hash,
	})
	if err != nil {
		log.Printf("Unable to get content: %v", err)
	}

	os.WriteFile("testdata/test.txt", []byte(content), 0644)

	contentChan := make(chan []byte)

	go func() {
		defer close(contentChan)
		contentChan <- []byte(content)
	}()

	computedHash, err := client.StoreContent(contentChan, hash, struct {
		RoutingKey string
	}{
		RoutingKey: hash,
	})
	if err != nil {
		log.Fatalf("Unable to store content: %v", err)
	}

	log.Printf("Stored content with hash: %v", hash)
	log.Printf("Computed hash: %v", computedHash)

	exists, err := client.IsCachedNearby(hash)
	if err != nil {
		log.Fatalf("Unable to check if content is cached nearby: %v", err)
	}

	log.Printf("Content is cached nearby: %v", exists)

	gotContent, err = client.GetContent(hash, 0, int64(len(content)), struct {
		RoutingKey string
	}{
		RoutingKey: hash,
	})
	if err != nil {
		log.Fatalf("Unable to get content: %v", err)
	}

	log.Printf("Got content: %v", string(gotContent))
}
