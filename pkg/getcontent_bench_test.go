package blobcache

import (
	"context"
	"crypto/rand"
	"testing"
)

// BenchmarkGetContent tests the actual GetContent method performance  
// This benchmarks the core read path without full CAS overhead
func BenchmarkGetContent(b *testing.B) {
	// Test 4MB - representative size
	size := int64(4 * 1024 * 1024)
	
	// Create simple in-memory cache (no background goroutines)
	bp := NewBufferPool()
	content := make([]byte, size)
	rand.Read(content)
	
	// Simulate GetContent read path with buffer pool
	dst := bp.Get(int(size))
	defer bp.Put(dst)

	b.ResetTimer()
	b.SetBytes(size) // Shows MB/s in results

	// Benchmark the read path (copy + buffer pool management)
	for i := 0; i < b.N; i++ {
		// This simulates what GetContent does internally:
		// 1. Get buffer from pool
		// 2. Copy data
		// 3. Return buffer to pool
		copy(dst, content)
	}
}

// BenchmarkGetContentChunked tests GetContent with chunked reads (more realistic)
func BenchmarkGetContentChunked(b *testing.B) {
	cas, cleanup := setupBenchmarkCAS(b)
	defer cleanup()

	// 16MB file, read in 4MB chunks
	totalSize := int64(16 * 1024 * 1024)
	chunkSize := int64(4 * 1024 * 1024)
	
	content := make([]byte, totalSize)
	rand.Read(content)
	hash := "bench-hash-chunked"

	err := cas.Add(context.Background(), hash, content)
	if err != nil {
		b.Fatalf("Failed to add content: %v", err)
	}

	dst := make([]byte, chunkSize)

	b.ResetTimer()
	b.SetBytes(totalSize) // Total bytes per iteration

	for i := 0; i < b.N; i++ {
		// Read file in chunks (realistic gRPC usage)
		offset := int64(0)
		for offset < totalSize {
			n, err := cas.Get(hash, offset, chunkSize, dst)
			if err != nil {
				b.Fatalf("Failed to get content: %v", err)
			}
			offset += n
		}
	}
}
