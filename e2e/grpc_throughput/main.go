package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	proto "github.com/beam-cloud/blobcache-v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// E2E gRPC throughput benchmark
// Tests real server/client communication with various configurations

type TestConfig struct {
	ServerAddr          string
	NumFiles            int
	FileSizes           []int64
	ChunkSize           int64
	Iterations          int
	TestWrite           bool
	TestRead            bool
	TestStream          bool
	Concurrency         int
	InitialWindowSize   int
	InitialConnWindow   int
	MaxConcurrentStreams int
}

type TestResult struct {
	TestName        string
	Operation       string
	TotalBytes      int64
	Duration        time.Duration
	ThroughputMBps  float64
	IOPS            float64
	AvgLatencyMs    float64
	P50LatencyMs    float64
	P99LatencyMs    float64
	Success         bool
	Error           string
}

func main() {
	config := &TestConfig{}
	
	flag.StringVar(&config.ServerAddr, "server", "localhost:50051", "gRPC server address")
	flag.IntVar(&config.NumFiles, "files", 10, "Number of files to test")
	flag.Int64Var(&config.ChunkSize, "chunksize", 4*1024*1024, "Chunk size in bytes")
	flag.IntVar(&config.Iterations, "iterations", 3, "Number of iterations")
	flag.BoolVar(&config.TestWrite, "write", true, "Test write operations")
	flag.BoolVar(&config.TestRead, "read", true, "Test read operations")
	flag.BoolVar(&config.TestStream, "stream", true, "Test streaming operations")
	flag.IntVar(&config.Concurrency, "concurrency", 1, "Concurrent operations")
	outputFile := flag.String("output", "", "Output file for results (JSON)")
	flag.Parse()
	
	fmt.Println("========================================")
	fmt.Println(" gRPC Throughput Benchmark")
	fmt.Println("========================================")
	fmt.Printf("Server: %s\n", config.ServerAddr)
	fmt.Printf("Files: %d\n", config.NumFiles)
	fmt.Printf("Chunk Size: %d MB\n", config.ChunkSize/(1024*1024))
	fmt.Printf("Iterations: %d\n", config.Iterations)
	fmt.Printf("Concurrency: %d\n\n", config.Concurrency)
	
	// Define test file sizes
	config.FileSizes = []int64{
		1 * 1024 * 1024,      // 1 MB
		16 * 1024 * 1024,     // 16 MB
		64 * 1024 * 1024,     // 64 MB
		256 * 1024 * 1024,    // 256 MB
	}
	
	// Run tests with different configurations
	results := []TestResult{}
	
	// Test default configuration
	fmt.Println("=== Testing Default Configuration ===")
	config.InitialWindowSize = 4 * 1024 * 1024
	config.InitialConnWindow = 32 * 1024 * 1024
	config.MaxConcurrentStreams = 1024
	results = append(results, runTestSuite(config, "Default")...)
	
	// Test conservative configuration
	fmt.Println("\n=== Testing Conservative Configuration ===")
	config.InitialWindowSize = 64 * 1024
	config.InitialConnWindow = 256 * 1024
	config.MaxConcurrentStreams = 100
	results = append(results, runTestSuite(config, "Conservative")...)
	
	// Test aggressive configuration
	fmt.Println("\n=== Testing Aggressive Configuration ===")
	config.InitialWindowSize = 16 * 1024 * 1024
	config.InitialConnWindow = 64 * 1024 * 1024
	config.MaxConcurrentStreams = 2048
	results = append(results, runTestSuite(config, "Aggressive")...)
	
	// Print summary
	printSummary(results)
	
	// Save results if output file specified
	if *outputFile != "" {
		if err := saveResults(results, *outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save results: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nResults saved to: %s\n", *outputFile)
	}
}

func runTestSuite(config *TestConfig, configName string) []TestResult {
	results := []TestResult{}
	
	for _, fileSize := range config.FileSizes {
		sizeMB := fileSize / (1024 * 1024)
		
		if config.TestWrite {
			result := testWrite(config, fileSize, configName)
			result.TestName = fmt.Sprintf("%s_Write_%dMB", configName, sizeMB)
			results = append(results, result)
			printResult(result)
		}
		
		if config.TestRead {
			result := testRead(config, fileSize, configName)
			result.TestName = fmt.Sprintf("%s_Read_%dMB", configName, sizeMB)
			results = append(results, result)
			printResult(result)
		}
		
		if config.TestStream {
			result := testStream(config, fileSize, configName)
			result.TestName = fmt.Sprintf("%s_Stream_%dMB", configName, sizeMB)
			results = append(results, result)
			printResult(result)
		}
	}
	
	return results
}

func createClient(config *TestConfig) (proto.BlobCacheClient, *grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithInitialWindowSize(int32(config.InitialWindowSize)),
		grpc.WithInitialConnWindowSize(int32(config.InitialConnWindow)),
		grpc.WithWriteBufferSize(256 * 1024),
		grpc.WithReadBufferSize(256 * 1024),
	}
	
	conn, err := grpc.DialContext(ctx, config.ServerAddr, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}
	
	return proto.NewBlobCacheClient(conn), conn, nil
}

func testWrite(config *TestConfig, fileSize int64, configName string) TestResult {
	result := TestResult{
		Operation: "Write",
		Success:   false,
	}
	
	client, conn, err := createClient(config)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()
	
	// Generate test data
	data := make([]byte, fileSize)
	rand.Read(data)
	
	latencies := []float64{}
	totalBytes := int64(0)
	start := time.Now()
	
	for i := 0; i < config.Iterations; i++ {
		iterStart := time.Now()
		
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		stream, err := client.StoreContent(ctx)
		if err != nil {
			cancel()
			result.Error = fmt.Sprintf("failed to create stream: %v", err)
			return result
		}
		
		// Send data in chunks
		sent := int64(0)
		for sent < fileSize {
			chunkSize := config.ChunkSize
			if sent+chunkSize > fileSize {
				chunkSize = fileSize - sent
			}
			
			if err := stream.Send(&proto.StoreContentRequest{
				Content: data[sent : sent+chunkSize],
			}); err != nil {
				cancel()
				result.Error = fmt.Sprintf("failed to send chunk: %v", err)
				return result
			}
			
			sent += chunkSize
		}
		
		_, err = stream.CloseAndRecv()
		cancel()
		if err != nil {
			result.Error = fmt.Sprintf("failed to close stream: %v", err)
			return result
		}
		
		latency := time.Since(iterStart)
		latencies = append(latencies, float64(latency.Milliseconds()))
		totalBytes += fileSize
	}
	
	duration := time.Since(start)
	
	result.TotalBytes = totalBytes
	result.Duration = duration
	result.ThroughputMBps = float64(totalBytes) / (1024 * 1024) / duration.Seconds()
	result.IOPS = float64(config.Iterations) / duration.Seconds()
	result.AvgLatencyMs = avg(latencies)
	result.P50LatencyMs = percentile(latencies, 50)
	result.P99LatencyMs = percentile(latencies, 99)
	result.Success = true
	
	return result
}

func testRead(config *TestConfig, fileSize int64, configName string) TestResult {
	result := TestResult{
		Operation: "Read",
		Success:   false,
	}
	
	client, conn, err := createClient(config)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()
	
	// First, store test data
	data := make([]byte, fileSize)
	rand.Read(data)
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	stream, err := client.StoreContent(ctx)
	if err != nil {
		cancel()
		result.Error = fmt.Sprintf("setup failed: %v", err)
		return result
	}
	
	stream.Send(&proto.StoreContentRequest{Content: data})
	resp, err := stream.CloseAndRecv()
	cancel()
	if err != nil {
		result.Error = fmt.Sprintf("setup failed: %v", err)
		return result
	}
	hash := resp.Hash
	
	// Now test reads
	latencies := []float64{}
	totalBytes := int64(0)
	start := time.Now()
	
	for i := 0; i < config.Iterations; i++ {
		iterStart := time.Now()
		
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		getResp, err := client.GetContent(ctx, &proto.GetContentRequest{
			Hash:   hash,
			Offset: 0,
			Length: fileSize,
		})
		cancel()
		
		if err != nil || !getResp.Ok {
			result.Error = fmt.Sprintf("read failed: %v", err)
			return result
		}
		
		latency := time.Since(iterStart)
		latencies = append(latencies, float64(latency.Milliseconds()))
		totalBytes += int64(len(getResp.Content))
	}
	
	duration := time.Since(start)
	
	result.TotalBytes = totalBytes
	result.Duration = duration
	result.ThroughputMBps = float64(totalBytes) / (1024 * 1024) / duration.Seconds()
	result.IOPS = float64(config.Iterations) / duration.Seconds()
	result.AvgLatencyMs = avg(latencies)
	result.P50LatencyMs = percentile(latencies, 50)
	result.P99LatencyMs = percentile(latencies, 99)
	result.Success = true
	
	return result
}

func testStream(config *TestConfig, fileSize int64, configName string) TestResult {
	result := TestResult{
		Operation: "Stream",
		Success:   false,
	}
	
	client, conn, err := createClient(config)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer conn.Close()
	
	// Store test data first
	data := make([]byte, fileSize)
	rand.Read(data)
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	stream, err := client.StoreContent(ctx)
	if err != nil {
		cancel()
		result.Error = fmt.Sprintf("setup failed: %v", err)
		return result
	}
	
	stream.Send(&proto.StoreContentRequest{Content: data})
	resp, err := stream.CloseAndRecv()
	cancel()
	if err != nil {
		result.Error = fmt.Sprintf("setup failed: %v", err)
		return result
	}
	hash := resp.Hash
	
	// Test streaming reads
	latencies := []float64{}
	totalBytes := int64(0)
	start := time.Now()
	
	for i := 0; i < config.Iterations; i++ {
		iterStart := time.Now()
		
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		streamResp, err := client.GetContentStream(ctx, &proto.GetContentRequest{
			Hash:   hash,
			Offset: 0,
			Length: fileSize,
		})
		if err != nil {
			cancel()
			result.Error = fmt.Sprintf("stream failed: %v", err)
			return result
		}
		
		received := int64(0)
		for {
			chunk, err := streamResp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				cancel()
				result.Error = fmt.Sprintf("recv failed: %v", err)
				return result
			}
			received += int64(len(chunk.Content))
		}
		cancel()
		
		latency := time.Since(iterStart)
		latencies = append(latencies, float64(latency.Milliseconds()))
		totalBytes += received
	}
	
	duration := time.Since(start)
	
	result.TotalBytes = totalBytes
	result.Duration = duration
	result.ThroughputMBps = float64(totalBytes) / (1024 * 1024) / duration.Seconds()
	result.IOPS = float64(config.Iterations) / duration.Seconds()
	result.AvgLatencyMs = avg(latencies)
	result.P50LatencyMs = percentile(latencies, 50)
	result.P99LatencyMs = percentile(latencies, 99)
	result.Success = true
	
	return result
}

func printResult(result TestResult) {
	status := "✓"
	if !result.Success {
		status = "✗"
	}
	
	fmt.Printf("%s %s: %.2f MB/s (%.2f IOPS, p50=%.2fms, p99=%.2fms)\n",
		status, result.TestName, result.ThroughputMBps, result.IOPS,
		result.P50LatencyMs, result.P99LatencyMs)
	
	if !result.Success {
		fmt.Printf("  Error: %s\n", result.Error)
	}
}

func printSummary(results []TestResult) {
	fmt.Println("\n========================================")
	fmt.Println(" Performance Summary")
	fmt.Println("========================================")
	
	byConfig := make(map[string][]TestResult)
	for _, r := range results {
		config := ""
		if len(r.TestName) > 0 {
			parts := []rune(r.TestName)
			for i, c := range parts {
				if c == '_' {
					config = string(parts[:i])
					break
				}
			}
		}
		byConfig[config] = append(byConfig[config], r)
	}
	
	for config, configResults := range byConfig {
		if len(configResults) == 0 {
			continue
		}
		
		fmt.Printf("\n%s Configuration:\n", config)
		
		var totalThroughput, totalIOPS float64
		var successCount int
		
		for _, r := range configResults {
			if r.Success {
				totalThroughput += r.ThroughputMBps
				totalIOPS += r.IOPS
				successCount++
			}
		}
		
		if successCount > 0 {
			fmt.Printf("  Avg Throughput: %.2f MB/s\n", totalThroughput/float64(successCount))
			fmt.Printf("  Avg IOPS: %.2f\n", totalIOPS/float64(successCount))
			fmt.Printf("  Success Rate: %d/%d (%.1f%%)\n",
				successCount, len(configResults),
				float64(successCount)/float64(len(configResults))*100)
		}
	}
	
	// Determine best configuration
	fmt.Println("\n========================================")
	fmt.Println(" Recommendations")
	fmt.Println("========================================")
	
	bestConfig := ""
	bestThroughput := 0.0
	
	for config, configResults := range byConfig {
		var totalThroughput float64
		var successCount int
		
		for _, r := range configResults {
			if r.Success {
				totalThroughput += r.ThroughputMBps
				successCount++
			}
		}
		
		if successCount > 0 {
			avgThroughput := totalThroughput / float64(successCount)
			if avgThroughput > bestThroughput {
				bestThroughput = avgThroughput
				bestConfig = config
			}
		}
	}
	
	if bestConfig != "" {
		fmt.Printf("✓ Best Configuration: %s (%.2f MB/s average)\n", bestConfig, bestThroughput)
		
		// Print recommended settings based on best config
		switch bestConfig {
		case "Default":
			fmt.Println("\nRecommended Settings (already in use):")
			fmt.Println("  InitialWindowSize: 4 MB")
			fmt.Println("  InitialConnWindowSize: 32 MB")
			fmt.Println("  MaxConcurrentStreams: 1024")
		case "Conservative":
			fmt.Println("\nRecommended Settings:")
			fmt.Println("  InitialWindowSize: 64 KB")
			fmt.Println("  InitialConnWindowSize: 256 KB")
			fmt.Println("  MaxConcurrentStreams: 100")
		case "Aggressive":
			fmt.Println("\nRecommended Settings:")
			fmt.Println("  InitialWindowSize: 16 MB")
			fmt.Println("  InitialConnWindowSize: 64 MB")
			fmt.Println("  MaxConcurrentStreams: 2048")
		}
	}
}

func saveResults(results []TestResult, filename string) error {
	// Simple JSON-like output
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fmt.Fprintln(file, "[")
	for i, r := range results {
		fmt.Fprintf(file, "  {\n")
		fmt.Fprintf(file, "    \"test_name\": %q,\n", r.TestName)
		fmt.Fprintf(file, "    \"operation\": %q,\n", r.Operation)
		fmt.Fprintf(file, "    \"total_bytes\": %d,\n", r.TotalBytes)
		fmt.Fprintf(file, "    \"duration_ms\": %d,\n", r.Duration.Milliseconds())
		fmt.Fprintf(file, "    \"throughput_mbps\": %.2f,\n", r.ThroughputMBps)
		fmt.Fprintf(file, "    \"iops\": %.2f,\n", r.IOPS)
		fmt.Fprintf(file, "    \"avg_latency_ms\": %.2f,\n", r.AvgLatencyMs)
		fmt.Fprintf(file, "    \"p50_latency_ms\": %.2f,\n", r.P50LatencyMs)
		fmt.Fprintf(file, "    \"p99_latency_ms\": %.2f,\n", r.P99LatencyMs)
		fmt.Fprintf(file, "    \"success\": %t", r.Success)
		if r.Error != "" {
			fmt.Fprintf(file, ",\n    \"error\": %q\n", r.Error)
		} else {
			fmt.Fprintln(file)
		}
		if i < len(results)-1 {
			fmt.Fprintln(file, "  },")
		} else {
			fmt.Fprintln(file, "  }")
		}
	}
	fmt.Fprintln(file, "]")
	
	return nil
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// Simple percentile calculation (not sorting for performance)
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	// Bubble sort (good enough for small arrays)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	idx := (len(sorted) * p) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	
	return sorted[idx]
}
