package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

	filePath := "e2e/testclient/testdata/newfile.txt"
	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

	const chunkSize = 1024 * 1024 * 16 // 16MB chunks
	var totalTime float64

	for i := 0; i < 1; i++ {
		chunks := make(chan []byte, 1)

		// Read file in chunks and dump into channel for StoreContent RPC calls
		go func() {
			file, err := os.Open(filePath)
			if err != nil {
				log.Fatalf("err: %v\n", err)
			}
			defer file.Close()

			buf := make([]byte, chunkSize)
			for {
				n, err := file.Read(buf)
				if err != nil && err != io.EOF {
					log.Fatalf("err reading file: %v\n", err)
				}
				if n == 0 {
					break
				}
				chunks <- buf[:n]
			}

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

	_, err = client.StoreContentFromSource("images/test-key-5079899", 0)
	if err != nil {
		log.Fatalf("err storing content: %v\n", err)
	}

	if err != nil {
		log.Fatalf("err: %v\n", err)
	}

}
