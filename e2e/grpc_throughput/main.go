package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Simple gRPC throughput test - measures read/write performance

type TestResult struct {
	Operation      string  `json:"operation"`
	FileSize       string  `json:"file_size"`
	ThroughputMBps float64 `json:"throughput_mbps"`
	DurationMs     int64   `json:"duration_ms"`
	Success        bool    `json:"success"`
	Error          string  `json:"error,omitempty"`
}

func main() {
	serverAddr := flag.String("server", "localhost:50051", "gRPC server address")
	outputFile := flag.String("output", "", "Output file for results (JSON)")
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println(" gRPC Throughput Test")
	fmt.Println("========================================")
	fmt.Printf("Server: %s\n\n", *serverAddr)

	// Test with 4MB and 64MB files (representative sizes)
	tests := []struct {
		name string
		size int64
	}{
		{"4MB", 4 * 1024 * 1024},
		{"64MB", 64 * 1024 * 1024},
	}

	results := []TestResult{}

	for _, test := range tests {
		fmt.Printf("Testing %s files...\n", test.name)
		
		// Test write
		writeResult := testWrite(*serverAddr, test.name, test.size)
		results = append(results, writeResult)
		printResult(writeResult)

		// Test read (using hash from write)
		if writeResult.Success {
			readResult := testRead(*serverAddr, test.name, test.size, writeResult.Error) // Reusing Error field for hash
			results = append(results, readResult)
			printResult(readResult)
		}
		
		fmt.Println()
	}

	// Print summary
	fmt.Println("========================================")
	fmt.Println(" Summary")
	fmt.Println("========================================")
	totalTests := len(results)
	passed := 0
	for _, r := range results {
		if r.Success {
			passed++
		}
	}
	fmt.Printf("Tests: %d/%d passed\n", passed, totalTests)

	if passed < totalTests {
		fmt.Printf("⚠️  %d test(s) failed\n", totalTests-passed)
	}

	// Save results
	if *outputFile != "" {
		data, _ := json.MarshalIndent(results, "", "  ")
		os.WriteFile(*outputFile, data, 0644)
		fmt.Printf("\nResults saved to: %s\n", *outputFile)
	}

	if passed < totalTests {
		os.Exit(1)
	}
}

func createClient(serverAddr string) (proto.BlobCacheClient, *grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithInitialWindowSize(4 * 1024 * 1024),
		grpc.WithInitialConnWindowSize(32 * 1024 * 1024),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(256*1024*1024),
			grpc.MaxCallSendMsgSize(256*1024*1024),
		),
	}

	conn, err := grpc.DialContext(ctx, serverAddr, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("connection failed: %w", err)
	}

	return proto.NewBlobCacheClient(conn), conn, nil
}

func testWrite(serverAddr, sizeName string, fileSize int64) TestResult {
	result := TestResult{
		Operation: "Write",
		FileSize:  sizeName,
		Success:   false,
	}

	client, conn, err := createClient(serverAddr)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()

	// Generate test data
	data := make([]byte, fileSize)
	rand.Read(data)

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.StoreContent(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("stream failed: %v", err)
		return result
	}

	// Send in 4MB chunks
	chunkSize := int64(4 * 1024 * 1024)
	sent := int64(0)
	for sent < fileSize {
		size := chunkSize
		if sent+size > fileSize {
			size = fileSize - sent
		}
		if err := stream.Send(&proto.StoreContentRequest{Content: data[sent : sent+size]}); err != nil {
			result.Error = fmt.Sprintf("send failed: %v", err)
			return result
		}
		sent += size
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		result.Error = fmt.Sprintf("close failed: %v", err)
		return result
	}

	duration := time.Since(start)
	result.DurationMs = duration.Milliseconds()
	result.ThroughputMBps = float64(fileSize) / (1024 * 1024) / duration.Seconds()
	result.Success = true
	result.Error = resp.Hash // Store hash for read test

	return result
}

func testRead(serverAddr, sizeName string, fileSize int64, hash string) TestResult {
	result := TestResult{
		Operation: "Read",
		FileSize:  sizeName,
		Success:   false,
	}

	client, conn, err := createClient(serverAddr)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.GetContent(ctx, &proto.GetContentRequest{
		Hash:   hash,
		Offset: 0,
		Length: fileSize,
	})

	if err != nil {
		result.Error = fmt.Sprintf("get failed: %v", err)
		return result
	}

	if !resp.Ok {
		result.Error = "content not found"
		return result
	}

	duration := time.Since(start)
	result.DurationMs = duration.Milliseconds()
	result.ThroughputMBps = float64(fileSize) / (1024 * 1024) / duration.Seconds()
	result.Success = true

	return result
}

func printResult(r TestResult) {
	status := "✗"
	if r.Success {
		status = "✓"
	}

	fmt.Printf("  %s %s %s: %.2f MB/s (%dms)\n",
		status, r.Operation, r.FileSize, r.ThroughputMBps, r.DurationMs)

	if !r.Success && r.Error != "" {
		fmt.Printf("    Error: %s\n", r.Error)
	}
}
