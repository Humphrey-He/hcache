# HCache Testing Framework

This directory contains benchmarks and regression tests for HCache. These tests are designed to ensure the performance and correctness of the cache implementation.

## Benchmarks

The benchmarks in `benchmark_test.go` measure the performance of HCache under various conditions:

### Basic Operations

- `BenchmarkCacheOperations`: Tests basic cache operations with different cache sizes and shard counts
  - `Get/Hit`: Measures cache hit performance
  - `Get/Miss`: Measures cache miss performance
  - `Set/New`: Measures performance of setting new keys
  - `Set/Existing`: Measures performance of updating existing keys
  - `Delete`: Measures key deletion performance
  - `Mixed/Read80Write20`: Measures performance with 80% reads, 20% writes
  - `Mixed/Read50Write50`: Measures performance with 50% reads, 50% writes
  - `Mixed/Read20Write80`: Measures performance with 20% reads, 80% writes
  - `ZipfianAccess`: Measures performance with a zipfian distribution (simulating real-world access patterns)

### Concurrency

- `BenchmarkConcurrency`: Tests cache performance with different concurrency levels (1-128 goroutines)

### Eviction Policies

- `BenchmarkEvictionPolicies`: Tests performance of different eviction policies
  - `lru`: Least Recently Used
  - `lfu`: Least Frequently Used
  - `fifo`: First In, First Out
  - `random`: Random eviction

### TTL Expiration

- `BenchmarkTTL`: Tests performance with different TTL values
  - `100ms`
  - `500ms`
  - `1s`
  - `10s`

### Hit Ratio

- `BenchmarkHitRatio`: Tests performance with different cache hit ratios
  - `10%`
  - `25%`
  - `50%`
  - `75%`
  - `90%`
  - `99%`

### Value Size

- `BenchmarkValueSize`: Tests performance with different value sizes
  - `64B`
  - `256B`
  - `1KB`
  - `4KB`
  - `16KB`
  - `64KB`

## Regression Tests

The regression tests in `regression_test.go` verify the correctness of HCache:

- `TestRegressionCacheConcurrency`: Tests that the cache behaves correctly under concurrent access
- `TestRegressionCacheExpiration`: Tests that the cache correctly expires entries
- `TestRegressionCacheEviction`: Tests that the cache correctly evicts entries when full

## Running the Tests

### Running All Tests

```bash
go test ./test -v
```

### Running Specific Tests

```bash
go test ./test -run TestRegressionCacheExpiration -v
```

### Running All Benchmarks

```bash
go test ./test -bench=. -benchmem
```

### Running Specific Benchmarks

```bash
go test ./test -bench=BenchmarkCacheOperations/Size=10000/Shards=16 -benchmem
```

### Benchmark Options

- `-benchtime=5s`: Run each benchmark for 5 seconds
- `-count=3`: Run each benchmark 3 times
- `-cpu=1,2,4,8`: Run benchmarks with different GOMAXPROCS values
- `-benchmem`: Show memory allocation statistics
- `-trace=trace.out`: Generate execution trace
- `-memprofile=mem.out`: Generate memory profile
- `-cpuprofile=cpu.out`: Generate CPU profile

## CI Integration

These tests are designed to be integrated with CI systems. Here's an example GitHub Actions workflow:

```yaml
name: Performance Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    - name: Run benchmarks
      run: go test ./test -bench=. -benchmem > benchmark.txt
    - name: Compare benchmarks
      uses: benchmark-action/github-action-benchmark@v1
      with:
        tool: 'go'
        output-file-path: benchmark.txt
        github-token: ${{ secrets.GITHUB_TOKEN }}
        auto-push: true
        alert-threshold: '200%'
        comment-on-alert: true
        fail-on-alert: true
``` 