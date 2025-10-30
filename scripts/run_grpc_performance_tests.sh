#!/bin/bash
set -e

# gRPC Performance Test Runner
# Starts blobcache server, runs performance tests, generates reports

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="$(dirname "$SCRIPT_DIR")"
RESULTS_DIR="${WORKSPACE_DIR}/performance-results"
BASELINE_FILE="${RESULTS_DIR}/baseline.json"
CURRENT_FILE="${RESULTS_DIR}/current.json"
REPORT_FILE="${RESULTS_DIR}/report.md"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SERVER_PORT=${SERVER_PORT:-50051}
TEST_ITERATIONS=${TEST_ITERATIONS:-3}
REGRESSION_THRESHOLD=${REGRESSION_THRESHOLD:-10} # 10% threshold

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE} gRPC Performance Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Create results directory
mkdir -p "$RESULTS_DIR"

# Step 1: Build binaries
echo -e "${YELLOW}[1/6]${NC} Building binaries..."
cd "$WORKSPACE_DIR"

if ! go build -o bin/blobcache cmd/main.go; then
    echo -e "${RED}✗ Failed to build blobcache${NC}"
    exit 1
fi

if ! go build -o bin/grpc-throughput e2e/grpc_throughput/main.go; then
    echo -e "${RED}✗ Failed to build grpc-throughput${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Binaries built successfully${NC}"
echo ""

# Step 2: Prepare test environment
echo -e "${YELLOW}[2/6]${NC} Preparing test environment..."

# Create temporary directories
TEST_DIR=$(mktemp -d)
DISK_CACHE_DIR="${TEST_DIR}/disk-cache"
mkdir -p "$DISK_CACHE_DIR"

# Create test configuration
TEST_CONFIG="${TEST_DIR}/config.yaml"
cat > "$TEST_CONFIG" << EOF
server:
  mode: coordinator
  diskCacheDir: ${DISK_CACHE_DIR}
  diskCacheMaxUsagePct: 90
  maxCachePct: 50
  pageSizeBytes: 4194304
  objectTtlS: 300
  s3DownloadConcurrency: 8
  s3DownloadChunkSize: 16777216
  metadata:
    mode: local

global:
  serverPort: ${SERVER_PORT}
  defaultLocality: local
  coordinatorHost: localhost:${SERVER_PORT}
  discoveryIntervalS: 10
  rttThresholdMilliseconds: 100
  hostStorageCapacityThresholdPct: 0.95
  grpcDialTimeoutS: 10
  grpcMessageSizeBytes: 67108864
  debugMode: false
  prettyLogs: false

client:
  token: ""
  minRetryLengthBytes: 1048576
  maxGetContentAttempts: 3
  nTopHosts: 3
  blobfs:
    enabled: false

metrics:
  pushIntervalS: 60
  url: ""
  username: ""
  password: ""
EOF

echo -e "${GREEN}✓ Test environment ready${NC}"
echo "  Config: $TEST_CONFIG"
echo "  Cache Dir: $DISK_CACHE_DIR"
echo ""

# Step 3: Start blobcache server
echo -e "${YELLOW}[3/6]${NC} Starting blobcache server..."

SERVER_LOG="${TEST_DIR}/server.log"
CONFIG_PATH="$TEST_CONFIG" ./bin/blobcache > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!

# Function to cleanup on exit
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    rm -rf "$TEST_DIR"
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}

trap cleanup EXIT INT TERM

# Wait for server to be ready
echo "  Waiting for server to start..."
sleep 3

# Check if server is running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo -e "${RED}✗ Server failed to start${NC}"
    echo "Server log:"
    cat "$SERVER_LOG"
    exit 1
fi

# Test server connectivity
MAX_RETRIES=10
RETRY=0
while [ $RETRY -lt $MAX_RETRIES ]; do
    if nc -z localhost $SERVER_PORT 2>/dev/null; then
        echo -e "${GREEN}✓ Server is running on port ${SERVER_PORT}${NC}"
        echo "  PID: $SERVER_PID"
        break
    fi
    RETRY=$((RETRY + 1))
    if [ $RETRY -eq $MAX_RETRIES ]; then
        echo -e "${RED}✗ Server did not start in time${NC}"
        exit 1
    fi
    sleep 1
done
echo ""

# Step 4: Run performance tests
echo -e "${YELLOW}[4/6]${NC} Running performance tests..."
echo "  This may take several minutes..."
echo ""

if ! ./bin/grpc-throughput \
    -server "localhost:${SERVER_PORT}" \
    -iterations "$TEST_ITERATIONS" \
    -output "$CURRENT_FILE"; then
    echo -e "${RED}✗ Performance tests failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Performance tests completed${NC}"
echo ""

# Step 5: Compare with baseline (if exists)
echo -e "${YELLOW}[5/6]${NC} Analyzing results..."

REGRESSION_DETECTED=0

if [ -f "$BASELINE_FILE" ]; then
    echo "  Comparing with baseline..."
    
    # Simple comparison (would use proper JSON parsing in production)
    BASELINE_AVG=$(grep -o '"throughput_mbps": [0-9.]*' "$BASELINE_FILE" | awk '{sum+=$2; count++} END {if(count>0) print sum/count; else print 0}')
    CURRENT_AVG=$(grep -o '"throughput_mbps": [0-9.]*' "$CURRENT_FILE" | awk '{sum+=$2; count++} END {if(count>0) print sum/count; else print 0}')
    
    if [ ! -z "$BASELINE_AVG" ] && [ ! -z "$CURRENT_AVG" ]; then
        # Calculate percentage change
        CHANGE=$(awk "BEGIN {printf \"%.2f\", (($CURRENT_AVG - $BASELINE_AVG) / $BASELINE_AVG) * 100}")
        
        echo ""
        echo "  Performance Comparison:"
        echo "    Baseline: ${BASELINE_AVG} MB/s"
        echo "    Current:  ${CURRENT_AVG} MB/s"
        echo "    Change:   ${CHANGE}%"
        echo ""
        
        # Check for regression
        IS_REGRESSION=$(awk "BEGIN {if ($CHANGE < -$REGRESSION_THRESHOLD) print 1; else print 0}")
        
        if [ "$IS_REGRESSION" = "1" ]; then
            echo -e "${RED}⚠ Performance regression detected! (${CHANGE}%)${NC}"
            REGRESSION_DETECTED=1
        elif [ $(awk "BEGIN {if ($CHANGE > $REGRESSION_THRESHOLD) print 1; else print 0}") = "1" ]; then
            echo -e "${GREEN}✓ Performance improvement! (+${CHANGE}%)${NC}"
        else
            echo -e "${GREEN}✓ Performance stable (${CHANGE}%)${NC}"
        fi
    fi
else
    echo "  No baseline found - saving current results as baseline"
    cp "$CURRENT_FILE" "$BASELINE_FILE"
fi
echo ""

# Step 6: Generate report
echo -e "${YELLOW}[6/6]${NC} Generating report..."

cat > "$REPORT_FILE" << 'EOF_REPORT'
# gRPC Performance Test Report

**Date:** $(date '+%Y-%m-%d %H:%M:%S')  
**Server Port:** ${SERVER_PORT}  
**Iterations:** ${TEST_ITERATIONS}

## Test Results

### Current Performance

EOF_REPORT

# Parse and format results
if [ -f "$CURRENT_FILE" ]; then
    echo "| Configuration | Operation | File Size | Throughput (MB/s) | IOPS | p50 Latency (ms) | p99 Latency (ms) | Status |" >> "$REPORT_FILE"
    echo "|--------------|-----------|-----------|-------------------|------|------------------|------------------|--------|" >> "$REPORT_FILE"
    
    # Simple parsing (would use jq in production)
    grep -o '"test_name": "[^"]*"' "$CURRENT_FILE" | cut -d'"' -f4 | while read test_name; do
        # Extract test details from the same index
        operation=$(grep -A 1 "\"test_name\": \"$test_name\"" "$CURRENT_FILE" | grep '"operation"' | cut -d'"' -f4)
        throughput=$(grep -A 2 "\"test_name\": \"$test_name\"" "$CURRENT_FILE" | grep '"throughput_mbps"' | grep -o '[0-9.]*' | head -1)
        iops=$(grep -A 3 "\"test_name\": \"$test_name\"" "$CURRENT_FILE" | grep '"iops"' | grep -o '[0-9.]*' | head -1)
        p50=$(grep -A 4 "\"test_name\": \"$test_name\"" "$CURRENT_FILE" | grep '"p50_latency_ms"' | grep -o '[0-9.]*' | head -1)
        p99=$(grep -A 5 "\"test_name\": \"$test_name\"" "$CURRENT_FILE" | grep '"p99_latency_ms"' | grep -o '[0-9.]*' | head -1)
        success=$(grep -A 6 "\"test_name\": \"$test_name\"" "$CURRENT_FILE" | grep '"success"' | grep -o 'true\|false' | head -1)
        
        # Parse test name
        config=$(echo "$test_name" | cut -d'_' -f1)
        op=$(echo "$test_name" | cut -d'_' -f2)
        size=$(echo "$test_name" | cut -d'_' -f3)
        
        status="✓"
        [ "$success" = "false" ] && status="✗"
        
        echo "| $config | $op | $size | $throughput | $iops | $p50 | $p99 | $status |" >> "$REPORT_FILE"
    done
fi

cat >> "$REPORT_FILE" << EOF_REPORT

### Summary Statistics

EOF_REPORT

# Add summary stats
if [ -f "$CURRENT_FILE" ]; then
    AVG_THROUGHPUT=$(grep -o '"throughput_mbps": [0-9.]*' "$CURRENT_FILE" | awk '{sum+=$2; count++} END {printf "%.2f", sum/count}')
    MAX_THROUGHPUT=$(grep -o '"throughput_mbps": [0-9.]*' "$CURRENT_FILE" | awk '{max=$2; for(i=1;i<=NF;i++) if($i>max) max=$i} END {print max}')
    AVG_P99=$(grep -o '"p99_latency_ms": [0-9.]*' "$CURRENT_FILE" | awk '{sum+=$2; count++} END {printf "%.2f", sum/count}')
    
    echo "- **Average Throughput:** ${AVG_THROUGHPUT} MB/s" >> "$REPORT_FILE"
    echo "- **Peak Throughput:** ${MAX_THROUGHPUT} MB/s" >> "$REPORT_FILE"
    echo "- **Average p99 Latency:** ${AVG_P99} ms" >> "$REPORT_FILE"
fi

if [ -f "$BASELINE_FILE" ] && [ ! -z "$BASELINE_AVG" ]; then
    cat >> "$REPORT_FILE" << EOF_REPORT

### Baseline Comparison

- **Baseline Throughput:** ${BASELINE_AVG} MB/s
- **Current Throughput:** ${CURRENT_AVG} MB/s
- **Change:** ${CHANGE}%

EOF_REPORT

    if [ "$REGRESSION_DETECTED" = "1" ]; then
        echo "⚠️ **Performance Regression Detected!**" >> "$REPORT_FILE"
    elif [ $(awk "BEGIN {if ($CHANGE > $REGRESSION_THRESHOLD) print 1; else print 0}") = "1" ]; then
        echo "✓ **Performance Improvement Detected!**" >> "$REPORT_FILE"
    else
        echo "✓ **Performance Stable**" >> "$REPORT_FILE"
    fi
fi

cat >> "$REPORT_FILE" << 'EOF_REPORT'

## Configuration Recommendations

Based on the test results, the optimal configuration is determined by comparing throughput across different settings:

- **Default Configuration:** 4MB window, 32MB connection window, 1024 streams
- **Conservative Configuration:** 64KB window, 256KB connection window, 100 streams
- **Aggressive Configuration:** 16MB window, 64MB connection window, 2048 streams

The test automatically identifies the best-performing configuration.

## Files

- **Current Results:** `current.json`
- **Baseline Results:** `baseline.json`
- **Server Log:** Available in test directory

EOF_REPORT

echo -e "${GREEN}✓ Report generated: ${REPORT_FILE}${NC}"
echo ""

# Display report
cat "$REPORT_FILE"

# Exit with error if regression detected
if [ "$REGRESSION_DETECTED" = "1" ]; then
    echo ""
    echo -e "${RED}========================================${NC}"
    echo -e "${RED} PERFORMANCE REGRESSION DETECTED${NC}"
    echo -e "${RED}========================================${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN} All Performance Tests Passed!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Results saved to: $RESULTS_DIR"
