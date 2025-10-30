# CI/CD Performance Testing Guide

Complete guide for the automated performance testing pipeline in blobcache.

---

## Overview

The CI/CD performance testing system provides:
- **Automated gRPC throughput tests** with real server/client
- **Unit benchmark validation** for all optimizations
- **Regression detection** against baseline
- **Configuration optimization** testing
- **Clear, actionable reports** with recommendations

---

## Pipeline Components

### 1. GitHub Actions Workflow

**Location**: `.github/workflows/performance-tests.yml`

**Triggers**:
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop`
- Nightly at 2 AM UTC (scheduled)
- Manual dispatch with custom parameters

**Jobs**:
1. **unit-benchmarks**: Runs Go benchmarks, validates buffer pool performance
2. **grpc-throughput-tests**: E2E tests with real server/client
3. **integration-tests**: Unit tests and validation suite
4. **performance-summary**: Aggregates results, generates summary

### 2. gRPC Throughput Test Tool

**Location**: `e2e/grpc_throughput/main.go`

**Features**:
- Tests write, read, and streaming operations
- Multiple file sizes (1MB, 16MB, 64MB, 256MB)
- Three configuration profiles:
  - **Default**: 4MB windows, 1024 streams (recommended)
  - **Conservative**: 64KB windows, 100 streams
  - **Aggressive**: 16MB windows, 2048 streams
- Automatic best configuration detection
- JSON output for CI/CD integration

**Usage**:
```bash
# Build
go build -o bin/grpc-throughput e2e/grpc_throughput/main.go

# Run against server
./bin/grpc-throughput \
  -server localhost:50051 \
  -iterations 3 \
  -output results.json
```

**Output Metrics**:
- Throughput (MB/s)
- IOPS
- p50/p99 latency (ms)
- Success rate

### 3. Test Orchestration Script

**Location**: `scripts/run_grpc_performance_tests.sh`

**Features**:
- Automatic server lifecycle management
- Configuration testing
- Baseline comparison
- Regression detection (configurable threshold)
- Markdown report generation

**Usage**:
```bash
# Run with defaults
./scripts/run_grpc_performance_tests.sh

# Custom configuration
TEST_ITERATIONS=5 \
REGRESSION_THRESHOLD=15 \
./scripts/run_grpc_performance_tests.sh
```

**Environment Variables**:
- `SERVER_PORT`: gRPC server port (default: 50051)
- `TEST_ITERATIONS`: Number of test iterations (default: 3)
- `REGRESSION_THRESHOLD`: Regression threshold % (default: 10)

---

## Test Coverage

### Unit Benchmarks

Tests all optimization components:

```
BenchmarkSequentialRead      # Large file sequential reads (1-256 MB)
BenchmarkRandomRead          # Random access patterns (4KB-512KB)
BenchmarkSmallFiles          # Small file fan-out (1-100 KB)
BenchmarkCacheHitRatios      # L0/L1 cache performance
BenchmarkPrefetcher          # Sequential detection effectiveness
BenchmarkBufferPool          # Allocation performance
```

**Performance Criteria**:
- Buffer pool: < 100ns per operation
- Sequential reads: > 500 MB/s
- Cache hit ratio: L0 > 70%

### gRPC Throughput Tests

**Test Matrix**:
```
Configuration × Operation × File Size
  3         ×     3      ×      4      = 36 tests
```

**Configurations**:
1. Default (4MB/32MB/1024)
2. Conservative (64KB/256KB/100)
3. Aggressive (16MB/64MB/2048)

**Operations**:
1. Write (client → server)
2. Read (server → client, single RPC)
3. Stream (server → client, streaming RPC)

**File Sizes**:
1. 1 MB (latency-sensitive)
2. 16 MB (balanced)
3. 64 MB (throughput-focused)
4. 256 MB (large file)

---

## Regression Detection

### How It Works

1. **Baseline Establishment**:
   - First run on `main` branch saves results as baseline
   - Baseline stored as artifact for 365 days
   - Updated on each merge to `main`

2. **Comparison**:
   - Each test run compares against baseline
   - Calculates average throughput change
   - Triggers warning if change exceeds threshold

3. **Thresholds**:
   ```
   Regression:  < -10% (default, configurable)
   Improvement: > +10%
   Stable:      -10% to +10%
   ```

### Example Regression Report

```markdown
## Baseline Comparison

- Baseline Throughput: 850.25 MB/s
- Current Throughput: 745.18 MB/s
- Change: -12.35%

⚠️ Performance Regression Detected!
```

---

## CI/CD Integration

### Pull Request Flow

1. **Trigger**: PR opened/updated
2. **Run Tests**: All 3 job types execute
3. **Generate Report**: Performance report commented on PR
4. **Check Status**: Pass/fail based on regression detection
5. **Review**: Developers review results before merge

### Main Branch Flow

1. **Trigger**: Push to `main`
2. **Run Tests**: Full test suite
3. **Save Baseline**: Current results become new baseline
4. **Artifact**: Baseline saved for future comparisons

### Nightly Flow

1. **Trigger**: Scheduled (2 AM UTC)
2. **Full Suite**: Comprehensive testing
3. **Report**: Summary emailed/posted
4. **Monitoring**: Long-term performance tracking

---

## Performance Reports

### Report Structure

```markdown
# gRPC Performance Test Report

**Date:** 2025-10-30 12:34:56
**Server Port:** 50051
**Iterations:** 3

## Test Results

### Current Performance

| Configuration | Operation | File Size | Throughput | IOPS | p50 | p99 | Status |
|--------------|-----------|-----------|------------|------|-----|-----|--------|
| Default      | Write     | 1MB       | 1250.45    | 125  | 7.8 | 12.3| ✓      |
| Default      | Read      | 1MB       | 1450.23    | 145  | 6.5 | 10.1| ✓      |
...

### Summary Statistics

- Average Throughput: 982.34 MB/s
- Peak Throughput: 1450.23 MB/s
- Average p99 Latency: 18.45 ms

### Baseline Comparison

- Change: +5.2% (improvement)
✓ Performance Stable

## Configuration Recommendations

Based on test results, **Default Configuration** is optimal:
- InitialWindowSize: 4 MB
- InitialConnWindowSize: 32 MB
- MaxConcurrentStreams: 1024
```

---

## Local Testing

### Quick Test

```bash
# Run validation suite
./scripts/validate_optimizations.sh
```

### Full gRPC Test

```bash
# Build everything
go build -o bin/blobcache cmd/main.go
go build -o bin/grpc-throughput e2e/grpc_throughput/main.go

# Start server (terminal 1)
CONFIG_PATH=config.yaml ./bin/blobcache

# Run tests (terminal 2)
./bin/grpc-throughput -server localhost:50051 -iterations 5
```

### Orchestrated Test

```bash
# Run complete test suite
./scripts/run_grpc_performance_tests.sh
```

**Output**:
- `performance-results/current.json` - Test results
- `performance-results/baseline.json` - Baseline (if exists)
- `performance-results/report.md` - Markdown report

---

## Troubleshooting

### Server Won't Start

**Issue**: Server fails to start in CI
**Solutions**:
1. Check port availability: `netstat -ln | grep 50051`
2. Review server logs: `cat /tmp/*/server.log`
3. Verify config: `cat /tmp/*/config.yaml`

### Tests Timeout

**Issue**: Tests exceed timeout
**Solutions**:
1. Reduce iterations: `TEST_ITERATIONS=1`
2. Increase timeout in workflow
3. Check server performance

### False Regressions

**Issue**: Inconsistent performance causing false alarms
**Solutions**:
1. Increase iterations for stability
2. Adjust threshold: `REGRESSION_THRESHOLD=15`
3. Run multiple times to establish baseline

### Network Issues

**Issue**: Connection refused errors
**Solutions**:
1. Wait longer for server: Increase sleep time in script
2. Check firewall rules
3. Verify server is listening: `nc -z localhost 50051`

---

## Configuration Tuning

### Optimal Settings (From Tests)

Based on comprehensive testing, these are the optimal defaults:

```yaml
# Server gRPC Settings (applied in code)
InitialWindowSize: 4 MB
InitialConnWindowSize: 32 MB
MaxConcurrentStreams: 1024
WriteBufferSize: 256 KB
ReadBufferSize: 256 KB
NumStreamWorkers: CPU_COUNT × 2

# FUSE Settings
maxBackgroundTasks: 512
maxWriteKB: 1024
maxReadAheadKB: 128

# Storage Settings
pageSizeBytes: 4194304  # 4MB
maxCachePct: 50-60
```

### When to Adjust

**Conservative** (64KB/256KB/100):
- Low-bandwidth networks
- High-latency connections
- Memory-constrained environments
- Many small file operations

**Aggressive** (16MB/64MB/2048):
- High-bandwidth LAN
- Low-latency networks
- Memory-rich environments
- Large file streaming workloads

---

## Metrics & Monitoring

### Key Performance Indicators

**Throughput**:
- Target: > 600 MB/s sequential
- Baseline: Current main branch average
- Alert: < -10% from baseline

**Latency**:
- p50: < 10ms (target)
- p99: < 50ms (target)
- Alert: > 100ms p99

**IOPS**:
- Small files: > 100 IOPS
- Large files: > 10 IOPS
- Alert: < 50% of baseline

### CI/CD Metrics Dashboard

GitHub Actions provides:
- Test duration trends
- Pass/fail rates
- Artifact size over time
- Job execution history

---

## Best Practices

### 1. Baseline Management

- **Update baseline** after verified performance improvements
- **Preserve history** using long retention (365 days)
- **Document changes** in commit messages

### 2. Test Stability

- **Run multiple iterations** (3-5 recommended)
- **Consistent environment** (same instance type in CI)
- **Avoid network variability** (use dedicated test infrastructure)

### 3. Regression Response

When regression detected:
1. **Verify**: Re-run tests to confirm
2. **Investigate**: Check recent changes
3. **Profile**: Use pprof if needed
4. **Fix**: Address bottleneck
5. **Validate**: Confirm fix before merge

### 4. Performance Improvements

When improving performance:
1. **Benchmark first**: Establish baseline
2. **Change one thing**: Isolate impact
3. **Measure again**: Verify improvement
4. **Update baseline**: Save new target

---

## Future Enhancements

Potential improvements to the testing system:

1. **Historical Tracking**:
   - Database for long-term metrics
   - Trend analysis and graphs
   - Performance over time visualization

2. **Extended Tests**:
   - Concurrent client testing
   - Mixed workload patterns
   - Sustained load tests

3. **Advanced Analysis**:
   - Automatic bottleneck detection
   - Configuration recommendation ML
   - Predictive regression detection

4. **Integration**:
   - Slack/email notifications
   - Grafana dashboards
   - PagerDuty alerting

---

## Support

### Resources

- **Test Results**: GitHub Actions artifacts
- **Reports**: `performance-results/report.md`
- **Logs**: Available in CI job output

### Common Commands

```bash
# List test runs
gh run list --workflow=performance-tests.yml

# Download artifacts
gh run download <run-id>

# View logs
gh run view <run-id> --log

# Trigger manual run
gh workflow run performance-tests.yml \
  -f test_iterations=5 \
  -f regression_threshold=15
```

---

## Summary

The CI/CD performance testing system:
- ✅ Automatically tests every PR
- ✅ Detects regressions before merge
- ✅ Validates optimization improvements
- ✅ Provides clear, actionable reports
- ✅ Tests multiple configurations
- ✅ Recommends optimal settings
- ✅ Tracks performance over time

**Result**: High-confidence deployment with continuous performance validation.

---

**Last Updated**: 2025-10-30  
**Version**: 1.0
