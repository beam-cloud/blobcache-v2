# CI/CD Performance Testing - Quick Start

Get started with the automated performance testing system in 5 minutes.

---

## 🚀 Quick Start (5 Minutes)

### 1. Build Everything

```bash
# Build blobcache server
go build -o bin/blobcache cmd/main.go

# Build gRPC throughput test
go build -o bin/grpc-throughput e2e/grpc_throughput/main.go

# Make scripts executable
chmod +x scripts/*.sh
```

### 2. Run Local Test

```bash
# Run complete test suite (starts server automatically)
./scripts/run_grpc_performance_tests.sh
```

**Output**:
```
========================================
 gRPC Performance Test Suite
========================================

[1/6] Building binaries...
✓ Binaries built successfully

[2/6] Preparing test environment...
✓ Test environment ready

[3/6] Starting blobcache server...
✓ Server is running on port 50051

[4/6] Running performance tests...
  This may take several minutes...

✓ Default_Write_1MB: 1250.45 MB/s (125 IOPS, p50=7.8ms, p99=12.3ms)
✓ Default_Read_1MB: 1450.23 MB/s (145 IOPS, p50=6.5ms, p99=10.1ms)
...

[5/6] Analyzing results...
  Performance stable (0.0%)

[6/6] Generating report...
✓ Report generated

========================================
 All Performance Tests Passed!
========================================

Results saved to: performance-results/
```

### 3. View Results

```bash
# View the report
cat performance-results/report.md

# View JSON results
cat performance-results/current.json

# Check for baseline
ls performance-results/baseline.json
```

---

## 📊 What Gets Tested

### Configuration Matrix

**3 Configurations** × **3 Operations** × **4 File Sizes** = **36 Tests**

**Configurations**:
1. **Default** (4MB windows, 1024 streams) ← Recommended
2. Conservative (64KB windows, 100 streams)
3. Aggressive (16MB windows, 2048 streams)

**Operations**:
1. Write (client → server)
2. Read (server → client)
3. Stream (server → client, streaming)

**File Sizes**:
1. 1 MB (latency test)
2. 16 MB (balanced)
3. 64 MB (throughput)
4. 256 MB (large file)

**Total Test Time**: ~5-10 minutes

---

## 🔄 CI/CD Integration

### Automatic Triggers

The performance tests run automatically on:
- ✅ Every push to `main` or `develop`
- ✅ Every pull request
- ✅ Nightly at 2 AM UTC
- ✅ Manual dispatch (on demand)

### What CI Does

1. **Unit Benchmarks**: Tests all optimizations
2. **gRPC Tests**: E2E with real server
3. **Integration Tests**: Full validation suite
4. **Report**: Posted as PR comment
5. **Regression Check**: Compares with baseline

### View CI Results

```bash
# List recent runs
gh run list --workflow=performance-tests.yml

# View specific run
gh run view <run-id>

# Download artifacts
gh run download <run-id>
```

---

## 📈 Understanding Results

### Good Performance

```
✓ Default_Read_64MB: 1250.45 MB/s (125 IOPS, p50=7.8ms, p99=12.3ms)
```

- Throughput: > 500 MB/s (sequential)
- p50 latency: < 10ms
- p99 latency: < 50ms
- Status: ✓ (passed)

### Performance Regression

```
⚠ Performance regression detected! (-12.35%)

Baseline: 850.25 MB/s
Current:  745.18 MB/s
Change:   -12.35%
```

**Action**: Investigate recent changes, fix bottleneck

### Performance Improvement

```
✓ Performance improvement! (+15.67%)

Baseline: 850.25 MB/s
Current:  983.47 MB/s
Change:   +15.67%
```

**Action**: Celebrate! 🎉 Baseline will be updated automatically.

---

## 🛠️ Customization

### Change Iterations

```bash
TEST_ITERATIONS=5 ./scripts/run_grpc_performance_tests.sh
```

More iterations = more stable results

### Change Threshold

```bash
REGRESSION_THRESHOLD=15 ./scripts/run_grpc_performance_tests.sh
```

Higher threshold = more tolerant of variance

### Custom Port

```bash
SERVER_PORT=9090 ./scripts/run_grpc_performance_tests.sh
```

Use different port if 50051 is busy

---

## 🐛 Troubleshooting

### Server Won't Start

```bash
# Check if port is in use
netstat -ln | grep 50051

# Kill existing server
pkill -f blobcache

# Try again
./scripts/run_grpc_performance_tests.sh
```

### Tests Fail

```bash
# More verbose output
set -x
./scripts/run_grpc_performance_tests.sh
```

### Inconsistent Results

```bash
# Run multiple times
for i in {1..3}; do
  echo "Run $i"
  ./scripts/run_grpc_performance_tests.sh
done
```

---

## 📁 File Locations

```
workspace/
├── e2e/
│   └── grpc_throughput/
│       └── main.go           ← gRPC test tool
├── scripts/
│   ├── run_grpc_performance_tests.sh  ← Test orchestration
│   └── validate_optimizations.sh      ← Quick validation
├── .github/
│   └── workflows/
│       └── performance-tests.yml      ← CI/CD pipeline
├── performance-results/       ← Test output (created on run)
│   ├── current.json          ← Latest results
│   ├── baseline.json         ← Baseline for comparison
│   └── report.md             ← Markdown report
└── Documentation:
    ├── CICD_QUICK_START.md   ← This file
    ├── CICD_PERFORMANCE_TESTING.md
    └── CICD_IMPLEMENTATION_SUMMARY.md
```

---

## ✅ Checklist

Before pushing to production:

- [x] Local test passes: `./scripts/run_grpc_performance_tests.sh`
- [x] Binaries build: `go build -o bin/* ...`
- [x] CI pipeline configured: `.github/workflows/performance-tests.yml`
- [ ] Baseline established (first run on main)
- [ ] Team trained on reading reports
- [ ] Regression threshold agreed upon
- [ ] Notification system configured (optional)

---

## 🎯 Next Steps

### This Week

1. Run local test: `./scripts/run_grpc_performance_tests.sh`
2. Review results in `performance-results/report.md`
3. Push changes to trigger CI
4. Monitor first CI run

### This Month

1. Establish stable baseline
2. Train team on CI reports
3. Set up notifications
4. Document any issues

### Ongoing

1. Monitor performance trends
2. Investigate regressions promptly
3. Update baseline on improvements
4. Refine thresholds as needed

---

## 📚 Documentation

- **Quick Start** (this file) - Get running fast
- **Complete Guide** (`CICD_PERFORMANCE_TESTING.md`) - Full details
- **Implementation Summary** (`CICD_IMPLEMENTATION_SUMMARY.md`) - Technical overview

---

## 💡 Pro Tips

1. **Run locally first** before pushing to CI
2. **Multiple iterations** give stable results
3. **Baseline on main** establishes truth
4. **Monitor trends** over time
5. **Document changes** that affect performance

---

## 🆘 Getting Help

### Common Questions

**Q: How long do tests take?**  
A: 5-10 minutes locally, similar in CI

**Q: What if I get different results?**  
A: Run multiple times, increase iterations

**Q: When should I update baseline?**  
A: Automatically updated on merge to main

**Q: Can I run just one configuration?**  
A: Modify `e2e/grpc_throughput/main.go` to skip configs

### Resources

- Test output: `performance-results/`
- Server logs: Shown in output
- CI logs: GitHub Actions page
- Documentation: `CICD_*.md` files

---

**Ready to test? Run this now:**

```bash
./scripts/run_grpc_performance_tests.sh
```

🚀 Happy testing!
