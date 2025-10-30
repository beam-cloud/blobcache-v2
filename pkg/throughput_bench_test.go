package blobcache

import (
	"crypto/rand"
	"testing"
)

// BenchmarkThroughputBaseline shows baseline read throughput WITHOUT buffer pool
// This simulates the old code path with allocations on every read
func BenchmarkThroughputBaseline(b *testing.B) {
	fileSize := int64(16 * 1024 * 1024) // 16 MB file
	chunkSize := int64(4 * 1024 * 1024) // 4 MB chunks
	
	// Prepare file data
	content := make([]byte, fileSize)
	rand.Read(content)

	b.ResetTimer()
	b.SetBytes(fileSize) // Reports MB/s for full file

	for i := 0; i < b.N; i++ {
		offset := int64(0)
		for offset < fileSize {
			readSize := chunkSize
			if offset+readSize > fileSize {
				readSize = fileSize - offset
			}

			// OLD WAY: Allocate new buffer each time (expensive!)
			dst := make([]byte, readSize)
			copy(dst, content[offset:offset+readSize])

			offset += readSize
		}
	}
}

// BenchmarkThroughputOptimized shows read throughput WITH buffer pool
// This simulates the new code path with buffer reuse
func BenchmarkThroughputOptimized(b *testing.B) {
	fileSize := int64(16 * 1024 * 1024) // 16 MB file  
	chunkSize := int64(4 * 1024 * 1024) // 4 MB chunks
	
	// Prepare file data
	content := make([]byte, fileSize)
	rand.Read(content)

	// NEW WAY: Use buffer pool
	pool := NewBufferPool()

	b.ResetTimer()
	b.SetBytes(fileSize) // Reports MB/s for full file

	for i := 0; i < b.N; i++ {
		offset := int64(0)
		for offset < fileSize {
			readSize := chunkSize
			if offset+readSize > fileSize {
				readSize = fileSize - offset
			}

			// Get buffer from pool (fast!)
			dst := pool.Get(int(readSize))
			copy(dst, content[offset:offset+readSize])
			pool.Put(dst) // Return to pool

			offset += readSize
		}
	}
}
