#!/bin/bash
set -e

# Validation script for blobcache optimizations
# This script runs a comprehensive test suite to validate performance improvements

echo "========================================="
echo " Blobcache Optimization Validation Suite"
echo "========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
print_success() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# 1. Build tests
echo "=== Step 1: Building Blobcache ==="
if go build -o bin/blobcache cmd/main.go; then
    print_success "Build successful"
else
    print_error "Build failed"
    exit 1
fi
echo ""

# 2. Unit tests
echo "=== Step 2: Running Unit Tests ==="
if go test -v ./pkg/...; then
    print_success "Unit tests passed"
else
    print_error "Unit tests failed"
fi
echo ""

# 3. Run benchmarks
echo "=== Step 3: Running Performance Benchmarks ==="
print_info "This may take several minutes..."
echo ""

BENCH_OUTPUT=$(mktemp)

# Sequential read benchmarks
echo "--- Sequential Read Benchmarks ---"
if go test -bench=BenchmarkSequentialRead -benchtime=5s -benchmem ./pkg/ | tee "$BENCH_OUTPUT"; then
    print_success "Sequential read benchmarks completed"
    
    # Extract and display key metrics
    echo ""
    echo "Key Metrics:"
    grep "MB/s" "$BENCH_OUTPUT" | tail -3
else
    print_error "Sequential read benchmarks failed"
fi
echo ""

# Random read benchmarks
echo "--- Random Read Benchmarks ---"
if go test -bench=BenchmarkRandomRead -benchtime=5s -benchmem ./pkg/ | tee "$BENCH_OUTPUT"; then
    print_success "Random read benchmarks completed"
else
    print_error "Random read benchmarks failed"
fi
echo ""

# Small file benchmarks
echo "--- Small File Benchmarks ---"
if go test -bench=BenchmarkSmallFiles -benchtime=5s -benchmem ./pkg/ | tee "$BENCH_OUTPUT"; then
    print_success "Small file benchmarks completed"
else
    print_error "Small file benchmarks failed"
fi
echo ""

# Cache hit ratio benchmarks
echo "--- Cache Hit Ratio Benchmarks ---"
if go test -bench=BenchmarkCacheHitRatios -benchtime=5s -benchmem ./pkg/ | tee "$BENCH_OUTPUT"; then
    print_success "Cache hit ratio benchmarks completed"
else
    print_error "Cache hit ratio benchmarks failed"
fi
echo ""

# Prefetcher benchmarks
echo "--- Prefetcher Benchmarks ---"
if go test -bench=BenchmarkPrefetcher -benchtime=5s -benchmem ./pkg/ | tee "$BENCH_OUTPUT"; then
    print_success "Prefetcher benchmarks completed"
else
    print_error "Prefetcher benchmarks failed"
fi
echo ""

# Buffer pool benchmarks
echo "--- Buffer Pool Benchmarks ---"
if go test -bench=BenchmarkBufferPool -benchtime=5s -benchmem ./pkg/ | tee "$BENCH_OUTPUT"; then
    print_success "Buffer pool benchmarks completed"
    
    # Show allocation improvements
    echo ""
    echo "Allocation Comparison:"
    grep "allocs/op" "$BENCH_OUTPUT" | tail -4
else
    print_error "Buffer pool benchmarks failed"
fi
echo ""

rm -f "$BENCH_OUTPUT"

# 4. Code quality checks
echo "=== Step 4: Code Quality Checks ==="

# Check for race conditions
print_info "Checking for race conditions..."
if go test -race ./pkg/...; then
    print_success "No race conditions detected"
else
    print_error "Race conditions found"
fi
echo ""

# Run go vet
print_info "Running go vet..."
if go vet ./...; then
    print_success "go vet passed"
else
    print_error "go vet found issues"
fi
echo ""

# Check formatting
print_info "Checking code formatting..."
UNFORMATTED=$(gofmt -l pkg/)
if [ -z "$UNFORMATTED" ]; then
    print_success "All code is properly formatted"
else
    print_error "Unformatted code found:"
    echo "$UNFORMATTED"
fi
echo ""

# 5. Verify new features exist
echo "=== Step 5: Verifying Implementation ==="

# Check for buffer pool
if grep -q "type BufferPool struct" pkg/buffer_pool.go 2>/dev/null; then
    print_success "Buffer pool implemented"
else
    print_error "Buffer pool not found"
fi

# Check for prefetcher
if grep -q "type Prefetcher struct" pkg/prefetcher.go 2>/dev/null; then
    print_success "Prefetcher implemented"
else
    print_error "Prefetcher not found"
fi

# Check for fadvise
if grep -q "fadviseSequential" pkg/fadvise.go 2>/dev/null; then
    print_success "fadvise support implemented"
else
    print_error "fadvise support not found"
fi

# Check for enhanced metrics
if grep -q "L0HitRatio" pkg/metrics.go 2>/dev/null; then
    print_success "Enhanced metrics implemented"
else
    print_error "Enhanced metrics not found"
fi

# Check for gRPC tuning
if grep -q "InitialWindowSize" pkg/server.go 2>/dev/null; then
    print_success "gRPC window tuning implemented"
else
    print_error "gRPC window tuning not found"
fi

# Check for FUSE optimizations
if grep -q "NegativeTimeout" pkg/blobfs.go 2>/dev/null; then
    print_success "FUSE negative caching implemented"
else
    print_error "FUSE negative caching not found"
fi

if grep -q "CongestionThreshold" pkg/blobfs.go 2>/dev/null; then
    print_success "FUSE congestion threshold implemented"
else
    print_error "FUSE congestion threshold not found"
fi

echo ""

# 6. Build E2E test
echo "=== Step 6: Building E2E Test ==="
if [ -f "e2e/throughput_bench/main.go" ]; then
    if go build -o bin/throughput-bench e2e/throughput_bench/main.go; then
        print_success "E2E throughput test built successfully"
        print_info "Run with: ./bin/throughput-bench -help"
    else
        print_error "E2E test build failed"
    fi
else
    print_error "E2E test source not found"
fi
echo ""

# Summary
echo "========================================="
echo " Validation Summary"
echo "========================================="
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All validations passed! ✓${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Review OPTIMIZATION_REPORT.md for details"
    echo "2. Run E2E tests: ./bin/throughput-bench"
    echo "3. Monitor metrics in your test environment"
    echo "4. Load test with real workloads"
    exit 0
else
    echo -e "${RED}Some validations failed. Please review the errors above.${NC}"
    exit 1
fi
