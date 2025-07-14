# EVM Performance Benchmarking Guide

This guide explains how to benchmark and compare performance between the original cosmos/evm and the evmone-integrated version.

## Overview

Your codebase has been enhanced with comprehensive benchmarking tools to evaluate the performance impact of integrating evmone as the EVM interpreter. The benchmarking suite includes:

- **Core EVM Operations**: Transaction processing, state transitions
- **Token Operations**: ERC20 transfers, mints, approvals
- **Smart Contract Interactions**: Message calls, log emissions
- **Memory and Allocation Analysis**: Resource usage comparison

## Quick Start

### 1. Run Full Comparison Benchmark

Compare performance between original and evmone versions:

```bash
./scripts/benchmark_comparison.sh
```

This will:
- Switch between branches automatically
- Run comprehensive benchmarks on both versions
- Generate detailed comparison reports
- Restore your original branch when complete

### 2. Quick Performance Test

Test current branch performance quickly:

```bash
./scripts/quick_benchmark.sh
```

### 3. Custom Benchmark Configuration

```bash
# Custom duration and iteration count
./scripts/benchmark_comparison.sh --duration 60s --count 10

# Specify different branches
./scripts/benchmark_comparison.sh --original-branch main --evmone-branch feature/my-evmone

# Quick test with custom settings
./scripts/quick_benchmark.sh --duration 30s --count 5
```

## Available Benchmarks

### Core EVM Benchmarks

Located in `tests/integration/x/vm/`:

- **BenchmarkApplyTransaction**: Core transaction execution
- **BenchmarkApplyMessage**: Message processing efficiency
- **BenchmarkTokenTransfer**: ERC20 token transfers
- **BenchmarkTokenMint**: Token minting operations
- **BenchmarkMessageCall**: Contract-to-contract calls
- **BenchmarkEmitLogs**: Event emission performance

### Running Individual Benchmarks

```bash
# Run specific benchmark category
go test -bench=BenchmarkApply ./tests/integration/x/vm/ -benchtime=30s

# Run with memory profiling
go test -bench=BenchmarkToken ./tests/integration/x/vm/ -benchmem

# Run with CPU profiling
go test -bench=BenchmarkMessage ./tests/integration/x/vm/ -cpuprofile=cpu.prof
```

## Understanding Results

### Benchmark Output Format

```
BenchmarkTokenTransfer-8    1000    1234567 ns/op    4096 B/op    32 allocs/op
```

- **BenchmarkTokenTransfer-8**: Test name and CPU count
- **1000**: Number of iterations
- **1234567 ns/op**: Nanoseconds per operation (lower is better)
- **4096 B/op**: Bytes allocated per operation (lower is better) 
- **32 allocs/op**: Number of allocations per operation (lower is better)

### Performance Metrics

- **Execution Time**: How fast operations complete
- **Memory Usage**: RAM consumption per operation
- **Allocations**: Number of memory allocations
- **Throughput**: Operations per second

## Evmone Integration Status

Codebase shows evmone integration is **active** with hardcoded configuration:

```go
// In evmd/app.go (lines 511-512)
useEVM1 := true
evm1Config := "lib/libevmone.0.15.0.dylib"
```

This means:
- ✅ Evmone library is integrated
- ✅ Using evmone v0.15.0
- ✅ Library located at `lib/libevmone.0.15.0.dylib`
- ✅ Integration is enabled by default

## Expected Performance Improvements

Based on evmone's design, you should expect:

### Likely Improvements ⚡
- **Smart Contract Execution**: 20-50% faster
- **Complex Operations**: Better for computationally intensive contracts
- **Memory Usage**: More efficient memory management
- **Gas Metering**: More accurate gas calculations

### Neutral/Variable 🔄
- **Simple Transfers**: Minimal difference for basic operations
- **State Access**: Similar performance for storage operations
- **Network I/O**: No impact on consensus or networking

### Potential Concerns ⚠️
- **Cold Start**: Possible slight increase in startup time
- **Memory Footprint**: Marginally higher baseline memory usage

## Interpreting Benchmark Results

### Significant Improvement (Good)
```
BenchmarkTokenTransfer: -25.3% time, -15.2% memory
```
- 25% faster execution
- 15% less memory usage
- Strong positive indicator for evmone

### Neutral Performance
```
BenchmarkTokenTransfer: +2.1% time, -1.5% memory  
```
- Minor variance within noise level
- No significant impact either way

### Performance Regression (Investigate)
```
BenchmarkTokenTransfer: +15.8% time, +10.2% memory
```
- Slower execution and higher memory usage
- May indicate integration issues or suboptimal configuration

## Advanced Analysis

### 1. Generate Detailed Reports

```bash
# Run full comparison and generate analysis
./scripts/benchmark_comparison.sh

# Analyze results with Python script
./scripts/analyze_benchmarks.py \
  --original benchmark_results/original_results.txt \
  --evmone benchmark_results/evmone_results.txt \
  --output benchmark_results/detailed_analysis.md
```

### 2. Profile Performance Bottlenecks

```bash
# CPU profiling
go test -bench=BenchmarkMessageCall ./tests/integration/x/vm/ -cpuprofile=cpu.prof

# Memory profiling  
go test -bench=BenchmarkTokenTransfer ./tests/integration/x/vm/ -memprofile=mem.prof

# View profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

### 3. Continuous Benchmarking

Add to CI/CD pipeline:

```yaml
- name: Run Performance Benchmarks
  run: |
    ./scripts/benchmark_comparison.sh --duration 30s --count 3
    if [ -f benchmark_results/comparison_report.txt ]; then
      echo "## Benchmark Results" >> $GITHUB_STEP_SUMMARY
      cat benchmark_results/comparison_report.txt >> $GITHUB_STEP_SUMMARY
    fi
```

## Troubleshooting

### Common Issues

**Build Failures**
```bash
# Ensure dependencies are up to date
go mod tidy
make build
```

**Permission Errors**
```bash
# Make scripts executable
chmod +x scripts/*.sh scripts/*.py
```

**Branch Not Found**
```bash
# List available branches
git branch -a

# Use custom branch names
./scripts/benchmark_comparison.sh --original-branch your-main --evmone-branch your-evmone
```

**Inconsistent Results**
- Increase benchmark duration: `--duration 60s`
- Increase iteration count: `--count 10` 
- Close other applications during benchmarking
- Run on dedicated hardware for consistent results

### Performance Debugging

If benchmarks show regressions:

1. **Verify Integration**: Check evmone library is loading correctly
2. **Check Configuration**: Ensure optimal evmone settings
3. **Profile Bottlenecks**: Use pprof to identify hot paths
4. **Test Isolation**: Run individual benchmarks to isolate issues
5. **Environment Consistency**: Ensure same hardware/OS for comparisons

## Best Practices

### 1. Benchmarking Environment
- Use dedicated hardware when possible
- Close unnecessary applications
- Run multiple iterations for statistical significance
- Benchmark on target deployment environment

### 2. Result Interpretation
- Focus on trends across multiple runs
- Consider both speed and memory metrics
- Evaluate real-world workload patterns
- Factor in development/maintenance costs

### 3. Performance Validation
- Complement benchmarks with integration tests
- Test with production-like workloads
- Validate gas calculation accuracy
- Monitor production metrics post-deployment

## Next Steps

1. **Run Initial Benchmarks**: Start with `./scripts/benchmark_comparison.sh`
2. **Analyze Results**: Review the generated reports in `benchmark_results/`
3. **Optimize Configuration**: Tune evmone settings based on results
4. **Validate Integration**: Ensure all tests pass with evmone enabled
5. **Plan Deployment**: Use benchmark data to inform rollout strategy

## Configuration Details

Your evmone integration uses these settings:

- **Library**: `lib/libevmone.0.15.0.dylib`
- **Version**: 0.15.0
- **Status**: Enabled by default
- **Fallback**: Original EVM available if needed

To modify the configuration, update the hardcoded values in `evmd/app.go` lines 511-512.

---

The benchmarking suite provides comprehensive performance analysis to validate your evmone integration. Use these tools to make data-driven decisions about the performance impact and optimize your EVM implementation.