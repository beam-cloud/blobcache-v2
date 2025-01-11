package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	cfg := configManager.GetConfig()

	// Initialize logger
	blobcache.InitLogger(cfg.DebugMode)

	ctx := context.Background()

	_, err = blobcache.NewBlobCacheClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Unable to create client: %v", err)
	}

	// Block until Ctrl+C (SIGINT) or SIGTERM is received
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Received interrupt or termination signal, exiting.")
}
