package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Throughput benchmark for validating blobcache read improvements
// Tests sequential and random read patterns with various file sizes

type BenchmarkConfig struct {
	MountPoint   string
	TestDataDir  string
	FileSize     int64
	ChunkSize    int64
	NumIterations int
	TestPattern  string // "sequential" or "random"
}

func main() {
	config := &BenchmarkConfig{}
	
	flag.StringVar(&config.MountPoint, "mount", "/tmp/blobcache", "FUSE mount point")
	flag.StringVar(&config.TestDataDir, "testdir", "/tmp/blobcache-test", "Test data directory")
	flag.Int64Var(&config.FileSize, "filesize", 256*1024*1024, "File size in bytes (default: 256MB)")
	flag.Int64Var(&config.ChunkSize, "chunksize", 4*1024*1024, "Read chunk size in bytes (default: 4MB)")
	flag.IntVar(&config.NumIterations, "iterations", 5, "Number of iterations")
	flag.StringVar(&config.TestPattern, "pattern", "sequential", "Test pattern: sequential or random")
	flag.Parse()
	
	fmt.Printf("=== Blobcache Throughput Benchmark ===\n")
	fmt.Printf("File Size: %d MB\n", config.FileSize/(1024*1024))
	fmt.Printf("Chunk Size: %d MB\n", config.ChunkSize/(1024*1024))
	fmt.Printf("Pattern: %s\n", config.TestPattern)
	fmt.Printf("Iterations: %d\n\n", config.NumIterations)
	
	// Create test file
	testFilePath := filepath.Join(config.TestDataDir, "benchmark-file.bin")
	if err := prepareTestFile(testFilePath, config.FileSize); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to prepare test file: %v\n", err)
		os.Exit(1)
	}
	
	// Run benchmark
	results := runBenchmark(config, testFilePath)
	
	// Print results
	printResults(results)
}

type BenchmarkResult struct {
	TotalBytes    int64
	Duration      time.Duration
	ThroughputMBs float64
	MinLatencyMs  float64
	MaxLatencyMs  float64
	AvgLatencyMs  float64
}

func prepareTestFile(path string, size int64) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Check if file already exists with correct size
	if stat, err := os.Stat(path); err == nil && stat.Size() == size {
		fmt.Printf("Using existing test file: %s\n", path)
		return nil
	}
	
	fmt.Printf("Creating test file: %s (%d MB)\n", path, size/(1024*1024))
	
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write random data in chunks
	chunkSize := int64(16 * 1024 * 1024) // 16MB chunks
	written := int64(0)
	
	for written < size {
		toWrite := chunkSize
		if written+toWrite > size {
			toWrite = size - written
		}
		
		chunk := make([]byte, toWrite)
		if _, err := rand.Read(chunk); err != nil {
			return err
		}
		
		if _, err := file.Write(chunk); err != nil {
			return err
		}
		
		written += toWrite
		
		if written%(100*1024*1024) == 0 {
			fmt.Printf("  Progress: %d MB / %d MB\n", written/(1024*1024), size/(1024*1024))
		}
	}
	
	return file.Sync()
}

func runBenchmark(config *BenchmarkConfig, testFilePath string) *BenchmarkResult {
	result := &BenchmarkResult{}
	
	var totalLatency float64
	result.MinLatencyMs = float64(^uint64(0) >> 1) // Max float64
	
	fmt.Printf("\nRunning benchmark...\n")
	
	overallStart := time.Now()
	
	for iter := 0; iter < config.NumIterations; iter++ {
		file, err := os.Open(testFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open test file: %v\n", err)
			os.Exit(1)
		}
		
		buffer := make([]byte, config.ChunkSize)
		offset := int64(0)
		chunkCount := 0
		
		for {
			// Read chunk
			start := time.Now()
			
			n, err := file.ReadAt(buffer, offset)
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
				break
			}
			
			if n == 0 {
				break
			}
			
			latency := time.Since(start)
			latencyMs := float64(latency.Microseconds()) / 1000.0
			
			totalLatency += latencyMs
			if latencyMs < result.MinLatencyMs {
				result.MinLatencyMs = latencyMs
			}
			if latencyMs > result.MaxLatencyMs {
				result.MaxLatencyMs = latencyMs
			}
			
			result.TotalBytes += int64(n)
			chunkCount++
			
			// Move to next chunk
			if config.TestPattern == "sequential" {
				offset += int64(n)
				if offset >= config.FileSize {
					break
				}
			} else {
				// Random pattern
				offset = (offset + config.ChunkSize*7) % (config.FileSize - config.ChunkSize)
			}
		}
		
		file.Close()
		
		iterDuration := time.Since(overallStart)
		iterThroughput := float64(result.TotalBytes) / (1024 * 1024) / iterDuration.Seconds()
		fmt.Printf("  Iteration %d: %.2f MB/s\n", iter+1, iterThroughput)
	}
	
	result.Duration = time.Since(overallStart)
	result.ThroughputMBs = float64(result.TotalBytes) / (1024 * 1024) / result.Duration.Seconds()
	
	if result.TotalBytes > 0 {
		chunkCount := result.TotalBytes / config.ChunkSize
		if chunkCount > 0 {
			result.AvgLatencyMs = totalLatency / float64(chunkCount)
		}
	}
	
	return result
}

func printResults(result *BenchmarkResult) {
	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("Total Data Read: %.2f MB\n", float64(result.TotalBytes)/(1024*1024))
	fmt.Printf("Duration: %.2f seconds\n", result.Duration.Seconds())
	fmt.Printf("Throughput: %.2f MB/s\n", result.ThroughputMBs)
	fmt.Printf("\nLatency Statistics:\n")
	fmt.Printf("  Min: %.2f ms\n", result.MinLatencyMs)
	fmt.Printf("  Avg: %.2f ms\n", result.AvgLatencyMs)
	fmt.Printf("  Max: %.2f ms\n", result.MaxLatencyMs)
}
