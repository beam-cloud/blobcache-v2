package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"time"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
)

var (
	totalIterations int  = 3
	checkHash       bool = false
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
		log.Fatalf("Unable to create client: %v\n", err)
	}

	filePath := "e2e/testclient/testdata/test3.bin"
	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Unable to read input file: %v\n", err)
	}
	hashBytes := sha256.Sum256(b)
	fileHash := hex.EncodeToString(hashBytes[:])

	const chunkSize = 1024 * 1024 * 16 // 16MB chunks
	var totalTime float64

	var storedHashed string = ""
	for i := 0; i < totalIterations; i++ {
		chunks := make(chan []byte)

		// Read file in chunks and dump into channel for StoreContent RPC calls
		go func() {
			file, err := os.Open(filePath)
			if err != nil {
				log.Fatalf("err: %v\n", err)
			}
			defer file.Close()

			for {
				buf := make([]byte, chunkSize)
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

		if storedHashed == "" {
			hash, err := client.StoreContent(chunks)
			if err != nil {
				log.Fatalf("Unable to store content: %v\n", err)
			}
			storedHashed = hash
		}

		startTime := time.Now()
		contentChan, err := client.GetContentStream(storedHashed, 0, int64(len(b)))
		if err != nil {
			log.Fatalf("Unable to get content stream: %v\n", err)
		}

		var content []byte
		chunkQueue := make(chan []byte, 10) // Buffered channel to queue chunks
		done := make(chan struct{})         // Channel to signal completion

		// Goroutine to write chunks to file and accumulate content
		go func() {
			file, err := os.Create("output.bin")
			if err != nil {
				log.Fatalf("Unable to create output file: %v\n", err)
			}
			defer file.Close()

			for chunk := range chunkQueue {
				_, err := file.Write(chunk)
				if err != nil {
					log.Fatalf("Error writing chunk to file: %v\n", err)
				}
				content = append(content, chunk...) // Accumulate chunks into content
			}
			close(done)
		}()

		for {
			chunk, ok := <-contentChan
			if !ok {
				break
			}
			chunkQueue <- chunk
		}
		close(chunkQueue) // Close the queue to signal no more chunks

		<-done // Wait for the file writing to complete
		elapsedTime := time.Since(startTime).Seconds()
		totalTime += elapsedTime

		if checkHash {
			hashBytes := sha256.Sum256(content)
			responseHash := hex.EncodeToString(hashBytes[:])

			log.Printf("Initial file len: %d\n", len(b))
			log.Printf("Response content len: %d\n", len(content))
			log.Printf("Hash of initial file: %s\n", fileHash)
			log.Printf("Hash of stored content: %s\n", storedHashed)
			log.Printf("Hash of retrieved content: %s\n", responseHash)
			log.Printf("Iteration %d: content length: %d, file length: %d, elapsed time: %f seconds\n", i+1, len(content), len(b), elapsedTime)

			if len(content) != len(b) {
				log.Fatalf("length mismatch: content len: %d, file len: %d\n", len(content), len(b))
			}

			// Direct byte comparison loop
			mismatchFound := false
			for i := range content {
				if content[i] != b[i] {
					log.Printf("Byte mismatch at position %d: content byte: %x, file byte: %x\n", i, content[i], b[i])
					mismatchFound = true
					break
				}
			}

			if !mismatchFound {
				log.Println("Direct byte comparison found no differences.")
			} else {
				log.Println("Direct byte comparison found differences.")
			}

			// Cross-check with bytes.Equal
			if bytes.Equal(content, b) {
				log.Println("bytes.Equal confirms the slices are equal.")
			} else {
				log.Println("bytes.Equal indicates the slices are not equal.")
			}

		}
	}

	averageTime := totalTime / float64(totalIterations)
	totalBytesReadMB := float64(len(b)*totalIterations) / (1024 * 1024)
	mbPerSecond := totalBytesReadMB / totalTime

	log.Printf("Total time: %f seconds\n", totalTime)
	log.Printf("Average time per iteration: %f seconds\n", averageTime)
	log.Printf("Total read: %.2f MB\n", totalBytesReadMB)
	log.Printf("Average MB/s rate of reading (GetContent): %f\n", mbPerSecond)
}
