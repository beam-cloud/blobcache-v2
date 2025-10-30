#!/bin/bash
# Baseline gRPC configuration test
# Tests different window sizes to find optimal settings

set -e

SERVER="localhost:50051"

echo "========================================="
echo " gRPC Configuration Baseline Test"
echo "========================================="
echo ""

# Test configurations
declare -A configs=(
    ["default"]="4194304 33554432"
    ["conservative"]="65536 262144"
    ["aggressive"]="16777216 67108864"
)

echo "Testing 3 configurations:"
echo "  1. Default: 4MB/32MB windows"
echo "  2. Conservative: 64KB/256KB windows"
echo "  3. Aggressive: 16MB/64MB windows"
echo ""

# Build test binary with config support
cat > /tmp/grpc_config_test.go << 'EOF'
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"time"
	proto "github.com/beam-cloud/blobcache-v2/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: prog <server> <window> <connwindow>")
		os.Exit(1)
	}
	
	server := os.Args[1]
	window, _ := strconv.Atoi(os.Args[2])
	connWindow, _ := strconv.Atoi(os.Args[3])
	
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithInitialWindowSize(int32(window)),
		grpc.WithInitialConnWindowSize(int32(connWindow)),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(256*1024*1024),
			grpc.MaxCallSendMsgSize(256*1024*1024),
		),
	}
	
	conn, err := grpc.Dial(server, opts...)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	
	client := proto.NewBlobCacheClient(conn)
	
	// Test 64MB transfer
	size := int64(64 * 1024 * 1024)
	data := make([]byte, size)
	rand.Read(data)
	
	start := time.Now()
	stream, _ := client.StoreContent(context.Background())
	chunkSize := int64(4 * 1024 * 1024)
	for i := int64(0); i < size; i += chunkSize {
		end := i + chunkSize
		if end > size {
			end = size
		}
		stream.Send(&proto.StoreContentRequest{Content: data[i:end]})
	}
	stream.CloseAndRecv()
	elapsed := time.Since(start)
	
	throughput := float64(size) / (1024 * 1024) / elapsed.Seconds()
	fmt.Printf("%.2f MB/s\n", throughput)
}
EOF

cd /workspace
go build -o /tmp/grpc_config_test /tmp/grpc_config_test.go 2>/dev/null

for name in "${!configs[@]}"; do
    IFS=' ' read -r window connwindow <<< "${configs[$name]}"
    result=$(/tmp/grpc_config_test "$SERVER" "$window" "$connwindow" 2>&1 || echo "Failed")
    echo "  $name (${window}/${connwindow}): $result"
done

rm -f /tmp/grpc_config_test /tmp/grpc_config_test.go

echo ""
echo "Use these results to tune grpcInitialWindowSize and grpcInitialConnWindowSize"
