#!/bin/bash
set -e

# Simple gRPC performance test - completes in under 1 minute

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="$(dirname "$SCRIPT_DIR")"
RESULTS_DIR="${WORKSPACE_DIR}/performance-results"
CURRENT_FILE="${RESULTS_DIR}/current.json"

echo "========================================"
echo " gRPC Performance Test"
echo "========================================"
echo ""

mkdir -p "$RESULTS_DIR"

# Build binaries
echo "[1/4] Building binaries..."
cd "$WORKSPACE_DIR"
go build -o bin/blobcache cmd/main.go
go build -o bin/grpc-throughput e2e/grpc_throughput/main.go
echo "✓ Build complete"
echo ""

# Setup test environment
echo "[2/4] Starting test server..."
TEST_DIR=$(mktemp -d)
DISK_CACHE_DIR="${TEST_DIR}/cache"
mkdir -p "$DISK_CACHE_DIR"

# Create minimal config
cat > "${TEST_DIR}/config.yaml" << EOF
server:
  mode: coordinator
  diskCacheDir: ${DISK_CACHE_DIR}
  diskCacheMaxUsagePct: 90
  maxCachePct: 50
  pageSizeBytes: 4194304
  metadata:
    mode: default
    redisAddr: "localhost:6379"

global:
  serverPort: 50051
  grpcMessageSizeBytes: 268435456
  debugMode: false

metrics:
  url: ""
EOF

# Start server
CONFIG_PATH="${TEST_DIR}/config.yaml" ./bin/blobcache > "${TEST_DIR}/server.log" 2>&1 &
SERVER_PID=$!

cleanup() {
    echo ""
    echo "Cleaning up..."
    if [ ! -z "$SERVER_PID" ]; then
        kill -9 $SERVER_PID 2>/dev/null || true
        sleep 1
    fi
    rm -rf "$TEST_DIR"
    echo "✓ Cleanup complete"
}
trap cleanup EXIT INT TERM

# Wait for server
sleep 3
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "✗ Server failed to start"
    cat "${TEST_DIR}/server.log"
    exit 1
fi

# Check connectivity
for i in {1..10}; do
    if nc -z localhost 50051 2>/dev/null; then
        echo "✓ Server ready"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "✗ Server not responding"
        exit 1
    fi
    sleep 1
done
echo ""

# Run tests
echo "[3/4] Running throughput tests..."
echo ""

./bin/grpc-throughput -server localhost:50051 -output "$CURRENT_FILE"
TEST_EXIT=$?

echo ""
if [ $TEST_EXIT -eq 0 ]; then
    echo "[4/4] ✓ All tests passed"
else
    echo "[4/4] ✗ Some tests failed"
fi

exit $TEST_EXIT
