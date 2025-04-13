package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"io"
	"log"
	"os"
	"time"

	blobcache "github.com/beam-cloud/blobcache-v2/pkg"
)

var (
	totalIterations int
	checkContent    bool
)

type TestResult struct {
	ElapsedTime        float64
	ContentCheckPassed bool
}

func main() {
	flag.IntVar(&totalIterations, "iterations", 3, "Number of iterations to run the tests")
	flag.BoolVar(&checkContent, "checkcontent", true, "Check the content hash after receiving data")
	flag.Parse()

	configManager, err := blobcache.NewConfigManager[blobcache.BlobCacheConfig]()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg := configManager.GetConfig().Client

	// Initialize logger
	blobcache.InitLogger(cfg.DebugMode, cfg.PrettyLogs)

	ctx := context.Background()

	client, err := blobcache.NewBlobCacheClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Unable to create client: %v", err)
	}

	filePath := "e2e/throughput/testdata/test3.bin"
	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Unable to read input file: %v\n", err)
	}
	hashBytes := sha256.Sum256(b)
	fileHash := hex.EncodeToString(hashBytes[:])

	hash, err := storeFile(client, filePath)
	if err != nil {
		log.Fatalf("Failed to store file: %v\n", err)
	}

	var totalStreamResult, totalGetContentResult TestResult
	for i := 0; i < totalIterations; i++ {
		log.Printf("Iteration %d\n", i)

		// Call GetContentStream
		log.Printf("TestGetContentStream - %v\n", hash)
		streamResult, err := TestGetContentStream(client, hash, len(b), fileHash)
		if err != nil {
			log.Fatalf("TestGetContentStream failed: %v\n", err)
		}
		totalStreamResult.ElapsedTime += streamResult.ElapsedTime
		totalStreamResult.ContentCheckPassed = totalStreamResult.ContentCheckPassed || streamResult.ContentCheckPassed
		log.Printf("TestGetContentStream - %v\n", streamResult)

		// Call GetContent
		log.Printf("TestGetContent - %v\n", hash)
		getContentResult, err := TestGetContent(client, hash, int64(len(b)), fileHash)
		if err != nil {
			log.Fatalf("TestGetContent failed: %v\n", err)
		}
		totalGetContentResult.ElapsedTime += getContentResult.ElapsedTime
		totalGetContentResult.ContentCheckPassed = totalGetContentResult.ContentCheckPassed || getContentResult.ContentCheckPassed
		log.Printf("TestGetContent - %v\n", getContentResult)
	}

	GenerateReport(totalStreamResult, totalGetContentResult, len(b), totalIterations)
}

func storeFile(client *blobcache.BlobCacheClient, filePath string) (string, error) {
	chunks := make(chan []byte)
	go func() {
		file, err := os.Open(filePath)
		if err != nil {
			log.Fatalf("err: %v\n", err)
		}
		defer file.Close()

		const chunkSize = 1024 * 1024 * 16 // 16MB chunks
		for {
			buf := make([]byte, chunkSize)
			n, err := file.Read(buf)

			if err != nil && err != io.EOF {
				log.Fatalf("err reading file: %v", err)
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
		return "", err
	}
	return hash, nil
}

func TestGetContentStream(client *blobcache.BlobCacheClient, hash string, fileSize int, expectedHash string) (TestResult, error) {
	contentCheckPassed := false

	startTime := time.Now()
	contentChan, err := client.GetContentStream(hash, 0, int64(fileSize))
	if err != nil {
		return TestResult{}, err
	}

	var contentStream []byte
	chunkQueue := make(chan []byte, 50) // Buffered channel to queue chunks
	done := make(chan struct{})         // Channel to signal completion

	go func() {
		for chunk := range chunkQueue {
			contentStream = append(contentStream, chunk...)
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
	<-done

	elapsedTime := time.Since(startTime).Seconds()

	// Verify received content's hash
	if checkContent {
		log.Printf("Verifying hash for GetContentStream")

		hashBytes := sha256.Sum256(contentStream)
		retrievedHash := hex.EncodeToString(hashBytes[:])
		if retrievedHash != expectedHash {
			contentCheckPassed = false
		} else {
			contentCheckPassed = true
		}

		log.Printf("Calculated hash for GetContentStream: expected %s, got %s", expectedHash, retrievedHash)
	}

	return TestResult{ElapsedTime: elapsedTime, ContentCheckPassed: contentCheckPassed}, nil
}

func TestGetContent(client *blobcache.BlobCacheClient, hash string, fileSize int64, expectedHash string) (TestResult, error) {
	startTime := time.Now()
	var content []byte
	offset := int64(0)
	const chunkSize = 1024 * 128 // 128k chunks

	for offset < fileSize {
		end := offset + chunkSize
		if end > fileSize {
			end = fileSize
		}

		chunk, err := client.GetContent(hash, offset, end-offset)
		if err != nil {
			return TestResult{}, err
		}
		content = append(content, chunk...)
		offset = end
	}

	elapsedTime := time.Since(startTime).Seconds()

	// Verify received content's hash
	contentCheckPassed := false
	if checkContent {
		log.Printf("Verifying hash for GetContent\n")

		hashBytes := sha256.Sum256(content)
		retrievedHash := hex.EncodeToString(hashBytes[:])
		if retrievedHash != expectedHash {
			contentCheckPassed = false
		} else {
			contentCheckPassed = true
		}

		log.Printf("Calculated hash for GetContent: expected %s, got %s", expectedHash, retrievedHash)
	}

	return TestResult{ElapsedTime: elapsedTime, ContentCheckPassed: contentCheckPassed}, nil
}

func GenerateReport(streamResult, contentResult TestResult, fileSize, iterations int) {
	averageTimeStream := streamResult.ElapsedTime / float64(iterations)
	averageTimeContent := contentResult.ElapsedTime / float64(iterations)
	totalBytesReadMB := float64(fileSize*iterations) / (1024 * 1024)
	mbPerSecondStream := totalBytesReadMB / streamResult.ElapsedTime
	mbPerSecondContent := totalBytesReadMB / contentResult.ElapsedTime

	log.Printf("Total time for GetContentStream: %f seconds\n", streamResult.ElapsedTime)
	log.Printf("Average time per iteration for GetContentStream: %f seconds\n", averageTimeStream)
	log.Printf("Total time for GetContent: %f seconds\n", contentResult.ElapsedTime)
	log.Printf("Average time per iteration for GetContent: %f seconds\n", averageTimeContent)
	log.Printf("Total read: %.2f MB\n", totalBytesReadMB)
	log.Printf("Average MB/s rate of reading (GetContentStream): %f\n", mbPerSecondStream)
	log.Printf("Average MB/s rate of reading (GetContent): %f\n", mbPerSecondContent)

	if checkContent {
		if streamResult.ContentCheckPassed && contentResult.ContentCheckPassed {
			log.Println("Content check passed for all iterations.")
		} else {
			log.Println("Content check failed for some iterations.")
		}
	}
}
