package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
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

	chunks := make(chan []byte, 1)

	b, err := os.ReadFile("e2e/testclient/testdata/test1.bin")
	if err != nil {
		fmt.Print(err)
	}

	// Read file in chunks and dump into channel for StoreContent RPC calls
	go func() {
		log.Printf("read %d bytes\n", len(b))
		chunks <- b[:]
		close(chunks)
	}()

	hash, err := client.StoreContent(chunks)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	content, err := client.GetContent(hash, 0, int64(len(b)))
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	log.Printf("content length: %d, file length: %d\n", len(content), len(b))
	fmt.Println(bytes.Compare(content, b)) // 0, means "equal"
}
