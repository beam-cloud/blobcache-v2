package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
)

func main() {
	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	cfg := configManager.GetConfig()

	// Initialize logger
	blobcache.InitLogger(cfg.DebugMode)

	ctx := context.Background()

	client, err := blobcache.NewBlobCacheClient(ctx, cfg)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	b, err := os.ReadFile("e2e/testclient/testdata/test1.bin")
	if err != nil {
		fmt.Print(err)
	}

	var totalTime float64
	for i := 0; i < 10; i++ {
		chunks := make(chan []byte, 1)

		// Read file in chunks and dump into channel for StoreContent RPC calls
		go func() {
			log.Printf("read %d bytes\n", len(b))
			chunks <- b[:]
			close(chunks)
		}()

		hash, err := client.StoreContent(chunks)
		if err != nil {
			log.Fatalf("err storing content: %v\n", err)
		}

		startTime := time.Now()
		content, err := client.GetContent(hash, 0, int64(len(b)))
		if err != nil {
			log.Fatalf("err getting content: %v\n", err)
		}
		elapsedTime := time.Since(startTime).Seconds()
		totalTime += elapsedTime

		log.Printf("Iteration %d: content length: %d, file length: %d, elapsed time: %f seconds\n", i+1, len(content), len(b), elapsedTime)
		fmt.Println(bytes.Compare(content, b)) // 0 means "equal"
	}

	averageTime := totalTime / 10
	mbPerSecond := (float64(len(b)) / (1024 * 1024)) / averageTime
	log.Printf("Average MB/s rate of reading (GetContent): %f\n", mbPerSecond)

}
