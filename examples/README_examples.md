# HCache Examples Guide

This document provides detailed guidance on using the example packages included in the HCache library. These examples demonstrate real-world applications, performance testing, and stress testing capabilities of HCache.

## Table of Contents

- [Benchmark Package](#benchmark-package)
- [HTTP Server Package](#http-server-package)
- [Stress Test Package](#stress-test-package)
- [中文版本](#中文版本)

## Benchmark Package

### Purpose

The benchmark package provides comprehensive performance testing tools for HCache. It helps developers:

- Evaluate the performance of different cache configurations
- Compare different eviction policies
- Understand how the cache behaves under various workloads
- Make informed decisions when configuring HCache for production

### Usage

To run all benchmarks:

```bash
cd examples/benchmark
go test -bench=. -benchmem
```

For more detailed analysis:

```bash
# Run each benchmark 3 times with 5s duration
go test -bench=. -benchmem -benchtime=5s -count=3

# Run specific benchmark
go test -bench=BenchmarkCache/Size=10000/Shards=16 -benchmem
```

### Key Metrics Explained

The benchmark outputs several important metrics:

- **ops/sec**: Operations per second, higher is better
- **ns/op**: Nanoseconds per operation, lower is better
- **B/op**: Bytes allocated per operation, lower is better
- **allocs/op**: Number of heap allocations per operation, lower is better

#### Interpreting Results

- **High ops/sec with low ns/op**: Indicates efficient cache operations
- **Low B/op and allocs/op**: Indicates good memory efficiency
- **Consistent performance across multiple runs**: Indicates stable behavior

### Important Considerations

1. **System Environment**: Benchmark results are influenced by hardware, OS, and Go version. When comparing results, ensure the environment is consistent.

2. **Realistic Workloads**: The benchmark includes tests with realistic access patterns (e.g., Zipfian distribution) that mimic real-world scenarios.

3. **Warmup Cycles**: Initial benchmark results may be affected by JIT compilation and other Go runtime optimizations. Use `-count=3` to get more stable results.

4. **Memory Profiling**: For deeper memory usage analysis, use:
   ```bash
   go test -bench=YourBenchmark -benchmem -memprofile=mem.prof
   go tool pprof mem.prof
   ```

## HTTP Server Package

### Purpose

The HTTP server package demonstrates how to integrate HCache with a web application. It implements a RESTful API for an e-commerce product catalog with proper caching strategies. This example shows:

- How to effectively use caching in a layered architecture
- Techniques for cache invalidation on data updates
- Methods for monitoring cache performance in a live application
- Best practices for error handling with cached data

### Usage

To run the HTTP server example:

```bash
cd examples/http_server
go run main.go
```

Configuration options:

```
-cache-size int        Maximum number of cache entries (default 1000)
-port string           HTTP server port (default "8080")
-shards int            Number of cache shards (default 16)
-ttl duration          Default TTL for cache entries (default 1m0s)
```

Once running, you can interact with the API using curl or any HTTP client:

```bash
# Get a product
curl http://localhost:8080/products/123

# Get cache statistics
curl http://localhost:8080/cache/stats
```

### Key Metrics and Monitoring

The `/cache/stats` endpoint provides important metrics:

- **Hit Rate**: Percentage of cache hits vs. total lookups
- **Hit Count**: Number of successful cache hits
- **Miss Count**: Number of cache misses
- **Eviction Count**: Number of entries removed due to capacity limits
- **Expiration Count**: Number of entries expired due to TTL
- **Entry Count**: Current number of items in the cache
- **Memory Usage**: Estimated memory consumption of the cache

#### Interpreting Results

- **High Hit Rate (>80%)**: Indicates effective cache utilization
- **High Eviction Count**: May indicate cache size is too small
- **High Expiration Count**: May indicate TTL is too short

### Important Considerations

1. **Cache Preloading**: The example demonstrates how to preload frequently accessed items for optimal performance.

2. **Cache Invalidation**: Updates to products trigger cache invalidation to prevent stale data.

3. **Error Handling**: The service layer gracefully handles cache failures by falling back to the primary data source.

4. **Concurrency**: The implementation is thread-safe and designed for high-concurrency environments.

5. **Memory Management**: Configure appropriate cache size based on your application's memory constraints.

## Stress Test Package

### Purpose

The stress test package allows you to simulate high-load scenarios and evaluate how HCache performs under pressure. It helps:

- Determine the maximum sustainable throughput
- Identify performance bottlenecks
- Test cache behavior under various read/write ratios
- Validate stability during prolonged high-load periods

### Usage

To run the stress test:

```bash
cd examples/stress_test
go run main.go
```

Configuration options:

```
-cache-size int         Maximum number of cache entries (default 100000)
-duration duration      Test duration (default 30s)
-keys int               Number of unique keys to use (default 10000)
-output string          Output format (text, csv, markdown) (default "text")
-qps int                Target QPS (default 1000)
-read-pct int           Percentage of read operations (vs writes) (default 80)
-report-interval        Interval for reporting stats (default 1s)
-shards int             Number of cache shards (default 16)
-ttl duration           Default TTL for cache entries (default 5m0s)
-value-size int         Size of values in bytes (default 1024)
-workers int            Number of concurrent workers (default 10)
```

### Key Metrics Explained

The stress test reports several metrics in real-time:

- **QPS**: Queries per second achieved (reads + writes)
- **Latency**: Response time distribution (min, avg, p95, p99, max)
- **Success Rate**: Percentage of operations completed without errors
- **Cache Hit Rate**: Percentage of read operations that hit the cache
- **Worker Saturation**: How busy the worker goroutines are

#### Interpreting Results

- **QPS vs Target QPS**: If actual QPS is below target, the system may be at capacity
- **Latency Increase**: Growing latency indicates performance degradation
- **Success Rate < 100%**: Indicates errors during operations
- **P99 Latency Spikes**: May indicate occasional garbage collection pauses

### Important Considerations

1. **Resource Monitoring**: Monitor system resources (CPU, memory) during the test to identify bottlenecks.

2. **Realistic Key Distribution**: The test supports different key access patterns to simulate realistic workloads.

3. **Warmup Period**: Allow a short warmup period before measuring performance metrics.

4. **Analyzing Results**: Use the CSV output option for post-test analysis with external tools.

5. **Test Duration**: Longer tests (10+ minutes) may reveal issues not apparent in short runs.

6. **Host Resource Contention**: Ensure the host system has sufficient resources available for accurate testing.

---

# 中文版本

# HCache 示例指南

本文档提供了 HCache 库中包含的示例包的详细使用指南。这些示例展示了 HCache 的实际应用、性能测试和压力测试能力。

## 目录

- [基准测试包](#基准测试包)
- [HTTP 服务器包](#http-服务器包)
- [压力测试包](#压力测试包)

## 基准测试包

### 用途

基准测试包为 HCache 提供了全面的性能测试工具。它可以帮助开发者：

- 评估不同缓存配置的性能表现
- 比较不同淘汰策略的效果
- 了解缓存在各种工作负载下的行为特性
- 为生产环境配置 HCache 时做出明智的决策

### 使用方法

运行所有基准测试：

```bash
cd examples/benchmark
go test -bench=. -benchmem
```

获取更详细的分析：

```bash
# 每个基准测试运行3次，每次持续5秒
go test -bench=. -benchmem -benchtime=5s -count=3

# 运行特定基准测试
go test -bench=BenchmarkCache/Size=10000/Shards=16 -benchmem
```

### 关键指标解读

基准测试输出几个重要的指标：

- **ops/sec**：每秒操作次数，越高越好
- **ns/op**：每次操作耗时（纳秒），越低越好
- **B/op**：每次操作分配的字节数，越低越好
- **allocs/op**：每次操作的堆分配次数，越低越好

#### 结果解读

- **高 ops/sec 和低 ns/op**：表示缓存操作效率高
- **低 B/op 和 allocs/op**：表示内存效率好
- **多次运行的性能一致**：表示行为稳定

### 注意事项

1. **系统环境**：基准测试结果受硬件、操作系统和 Go 版本影响。比较结果时，确保环境一致。

2. **真实工作负载**：基准测试包括具有真实访问模式的测试（如 Zipfian 分布），模拟真实场景。

3. **预热周期**：初始基准测试结果可能受到 JIT 编译和其他 Go 运行时优化的影响。使用 `-count=3` 获取更稳定的结果。

4. **内存分析**：对于更深入的内存使用分析，使用：
   ```bash
   go test -bench=YourBenchmark -benchmem -memprofile=mem.prof
   go tool pprof mem.prof
   ```

## HTTP 服务器包

### 用途

HTTP 服务器包演示了如何将 HCache 与 Web 应用程序集成。它实现了一个带有适当缓存策略的电子商务产品目录 RESTful API。这个示例展示了：

- 如何在分层架构中有效使用缓存
- 数据更新时的缓存失效技术
- 在实时应用程序中监控缓存性能的方法
- 处理缓存数据时的错误处理最佳实践

### 使用方法

运行 HTTP 服务器示例：

```bash
cd examples/http_server
go run main.go
```

配置选项：

```
-cache-size int        最大缓存条目数（默认 1000）
-port string           HTTP 服务器端口（默认 "8080"）
-shards int            缓存分片数（默认 16）
-ttl duration          缓存条目默认 TTL（默认 1m0s）
```

运行后，您可以使用 curl 或任何 HTTP 客户端与 API 交互：

```bash
# 获取产品
curl http://localhost:8080/products/123

# 获取缓存统计信息
curl http://localhost:8080/cache/stats
```

### 关键指标和监控

`/cache/stats` 端点提供重要指标：

- **命中率**：缓存命中次数占总查询次数的百分比
- **命中次数**：成功的缓存命中次数
- **未命中次数**：缓存未命中次数
- **淘汰次数**：由于容量限制而移除的条目数
- **过期次数**：由于 TTL 而过期的条目数
- **条目数**：当前缓存中的项目数
- **内存使用**：缓存的估计内存消耗

#### 结果解读

- **高命中率（>80%）**：表示缓存利用率高
- **高淘汰次数**：可能表示缓存大小过小
- **高过期次数**：可能表示 TTL 设置过短

### 注意事项

1. **缓存预加载**：示例演示了如何预加载频繁访问的项目以获得最佳性能。

2. **缓存失效**：产品更新会触发缓存失效，防止数据过时。

3. **错误处理**：服务层在缓存失败时会优雅地回退到主数据源。

4. **并发性**：实现是线程安全的，专为高并发环境设计。

5. **内存管理**：根据应用程序的内存限制配置适当的缓存大小。

## 压力测试包

### 用途

压力测试包允许您模拟高负载场景并评估 HCache 在压力下的性能表现。它有助于：

- 确定最大可持续吞吐量
- 识别性能瓶颈
- 测试不同读写比例下的缓存行为
- 验证在长时间高负载期间的稳定性

### 使用方法

运行压力测试：

```bash
cd examples/stress_test
go run main.go
```

配置选项：

```
-cache-size int         最大缓存条目数（默认 100000）
-duration duration      测试持续时间（默认 30s）
-keys int               使用的唯一键数量（默认 10000）
-output string          输出格式（text、csv、markdown）（默认 "text"）
-qps int                目标 QPS（默认 1000）
-read-pct int           读操作百分比（与写操作相比）（默认 80）
-report-interval        报告统计信息的间隔（默认 1s）
-shards int             缓存分片数（默认 16）
-ttl duration           缓存条目默认 TTL（默认 5m0s）
-value-size int         值大小（字节）（默认 1024）
-workers int            并发工作线程数（默认 10）
```

### 关键指标解读

压力测试实时报告几个指标：

- **QPS**：每秒查询次数（读 + 写）
- **延迟**：响应时间分布（最小值、平均值、P95、P99、最大值）
- **成功率**：无错误完成的操作百分比
- **缓存命中率**：命中缓存的读操作百分比
- **工作线程饱和度**：工作协程的忙碌程度

#### 结果解读

- **实际QPS与目标QPS**：如果实际QPS低于目标，系统可能已达到容量上限
- **延迟增加**：延迟增加表示性能下降
- **成功率<100%**：表示操作过程中出现错误
- **P99延迟峰值**：可能表示偶尔的垃圾收集暂停

### 注意事项

1. **资源监控**：在测试期间监控系统资源（CPU、内存）以识别瓶颈。

2. **真实键分布**：测试支持不同的键访问模式，以模拟真实工作负载。

3. **预热期**：在测量性能指标前，允许短暂的预热期。

4. **结果分析**：使用CSV输出选项进行测试后分析，配合外部工具。

5. **测试持续时间**：较长的测试（10+分钟）可能会揭示短时间运行中不明显的问题。

6. **主机资源竞争**：确保主机系统有足够的可用资源进行准确测试。 