# HCache 示例用法指南

本文档详细介绍了HCache提供的各种示例用法，包括如何运行、配置和解释结果。这些示例展示了HCache在不同场景下的应用方式和性能特性。

## 1. 基准测试示例 (Benchmark)

基准测试示例用于评估HCache在不同操作和配置下的性能。

### 1.1 示例位置

基准测试位于 `examples/benchmark` 目录下，主要文件是 `benchmark_test.go`。

### 1.2 运行方法

```bash
# 进入基准测试目录
cd examples/benchmark

# 运行所有基准测试
go test -bench=. -benchmem

# 运行特定配置的基准测试
go test -bench=BenchmarkCache/Size=1000/Shards=16 -benchmem

# 运行淘汰策略比较测试
go test -bench=BenchmarkEvictionPolicies -benchmem

# 运行TTL测试
go test -bench=BenchmarkTTL -benchmem

# 增加测试时间和重复次数以获得更可靠的结果
go test -bench=BenchmarkCache/Size=1000/Shards=16/Get/Hit -benchmem -benchtime=3s -count=3
```

### 1.3 配置参数说明

基准测试中的主要配置参数包括：

- **Size**：缓存大小，支持1000、10000、100000三种配置
- **Shards**：分片数量，支持1、4、16、64四种配置
- **操作类型**：
  - Get/Hit：测试缓存命中时的读取性能
  - Get/Miss：测试缓存未命中时的读取性能
  - Set/New：测试添加新键的性能
  - Set/Existing：测试更新现有键的性能
  - Mixed：测试混合读写负载(不同读写比例)
  - ZipfianAccess：测试在符合Zipfian分布的访问模式下的性能

### 1.4 结果解读

基准测试结果会显示每个操作的性能指标：

```
BenchmarkCache/Size=1000/Shards=16/Get/Hit-12   37124998    95.80 ns/op    0 B/op    0 allocs/op
```

- **37124998**：迭代次数，表示测试运行了多少次操作
- **95.80 ns/op**：每次操作的平均耗时(纳秒)
- **0 B/op**：每次操作的内存分配量(字节)
- **0 allocs/op**：每次操作的内存分配次数

对于淘汰策略测试，可以比较不同策略的性能差异；对于TTL测试，可以评估过期机制的开销。

### 1.5 注意事项

- 基准测试是单Goroutine运行的，不测试并发性能
- Zipfian测试在某些环境下可能存在越界问题，可以跳过此项测试
- 测试结果会受到机器负载和其他因素的影响，建议多次运行并取平均值

## 2. 压力测试示例 (Stress Test)

压力测试示例用于评估HCache在高并发和持续负载下的性能和稳定性。

### 2.1 示例位置

压力测试位于 `examples/stress_test` 目录下，主要文件是 `main.go`。

### 2.2 运行方法

```bash
# 进入压力测试目录
cd examples/stress_test

# 使用默认参数运行压力测试
go run main.go

# 自定义参数运行压力测试
go run main.go -qps=2000 -duration=15s -workers=8 -read-pct=80 -keys=5000
```

### 2.3 配置参数说明

压力测试的主要配置参数包括：

- **-qps**：目标每秒查询数(默认1000)
- **-duration**：测试持续时间(默认30s)
- **-workers**：并发工作线程数(默认10)
- **-read-pct**：读操作百分比(默认80)
- **-keys**：唯一键的数量(默认10000)
- **-value-size**：值的大小，单位字节(默认1024)
- **-cache-size**：缓存最大条目数(默认100000)
- **-ttl**：缓存条目默认TTL(默认5m)
- **-shards**：缓存分片数(默认16)
- **-output**：输出格式，支持text、csv、markdown(默认text)

### 2.4 结果解读

压力测试会每秒输出一次统计信息，包括：

```
[18:10:23] Requests: 2981 (990.01 req/s), Success: 100.00%, Reads: 80.21%, Writes: 19.79%, Hit rate: 6.19%
         Latency: avg=0.00 ms, p95=0.00 ms, max=0.05 ms
         Cache: entries=551, size=0 bytes
```

测试结束后会输出总结信息：

```
Test Results:
Duration: 15s
Total Requests: 14803
Successful Requests: 14803 (100.00%)
Failed Requests: 0 (0.00%)
Read Requests: 11848 (80.04%)
Write Requests: 2955 (19.96%)
Cache Hits: 2889
Cache Misses: 8959
Cache Hit Rate: 24.38%
Requests Per Second: 986.84
Average Latency: 0.00 ms
P95 Latency: 0.00 ms
Max Latency: 1.00 ms
```

这些指标可以帮助评估缓存在高负载下的性能和稳定性，特别是命中率、延迟和吞吐量。

### 2.5 注意事项

- 压力测试是多Goroutine并发运行的，可以测试并发性能
- 测试初期命中率通常较低，随着时间推移会逐渐提高
- 可以通过调整read-pct参数测试不同读写比例下的性能
- 在资源受限的环境下，可能无法达到目标QPS，此时会有警告信息

## 3. HTTP服务器示例 (HTTP Server)

HTTP服务器示例展示了如何在实际Web应用中集成HCache，实现API结果缓存。

### 3.1 示例位置

HTTP服务器示例位于 `examples/http_server` 目录下，入口文件是 `main.go`。

### 3.2 运行方法

```bash
# 进入HTTP服务器示例目录
cd examples/http_server

# 使用默认参数运行服务器
go run main.go

# 自定义参数运行服务器
go run main.go -port=8888 -cache-size=500 -ttl=30s
```

### 3.3 配置参数说明

HTTP服务器示例的主要配置参数包括：

- **-port**：HTTP服务器端口(默认8080)
- **-cache-size**：缓存最大条目数(默认1000)
- **-ttl**：缓存条目默认TTL(默认1m)
- **-shards**：缓存分片数(默认16)

### 3.4 API端点

服务器启动后，可以通过以下API端点进行测试：

- `GET /products/:id` - 获取单个产品
- `GET /products` - 列出产品(支持过滤)
- `POST /products` - 创建新产品
- `PUT /products/:id` - 更新产品
- `DELETE /products/:id` - 删除产品
- `GET /cache/stats` - 获取缓存统计信息

### 3.5 测试方法

可以使用curl或其他HTTP客户端工具测试API：

```bash
# 获取缓存统计信息
curl http://localhost:8080/cache/stats

# 获取产品列表
curl http://localhost:8080/products

# 获取单个产品
curl http://localhost:8080/products/1

# 创建新产品
curl -X POST -H "Content-Type: application/json" -d '{"name":"New Product","price":99.99,"category":"electronics"}' http://localhost:8080/products

# 更新产品
curl -X PUT -H "Content-Type: application/json" -d '{"name":"Updated Product","price":199.99,"category":"electronics"}' http://localhost:8080/products/1

# 删除产品
curl -X DELETE http://localhost:8080/products/1
```

### 3.6 缓存使用分析

HTTP服务器示例演示了几种常见的缓存模式：

1. **缓存旁路模式**：首先检查缓存，未命中时从存储中获取并添加到缓存
2. **预加载热点数据**：服务启动时预加载热门产品到缓存
3. **缓存失效**：在数据更新或删除时使相关缓存条目失效
4. **不同TTL策略**：为不同类型的数据设置不同的TTL

可以通过缓存统计API(`/cache/stats`)监控缓存的使用情况，包括命中率、条目数和内存使用。

### 3.7 注意事项

- 示例使用模拟存储，实际应用中应替换为真实数据库
- 缓存统计可以用于监控缓存效果，据此调整缓存大小和TTL
- 缓存预加载适合热点数据相对固定的场景
- 在高并发环境下，可以增加分片数以减少锁竞争

## 4. 自定义测试场景

除了提供的示例外，你还可以创建自定义测试场景来评估HCache在特定应用场景下的性能。

### 4.1 创建自定义基准测试

```go
func BenchmarkCustomScenario(b *testing.B) {
    // 创建缓存实例
    cacheInstance, err := cache.NewWithOptions("custom-cache",
        cache.WithMaxEntryCount(1000),
        cache.WithShards(16),
        cache.WithTTL(time.Minute),
        cache.WithEviction("lru"),
    )
    if err != nil {
        b.Fatalf("Failed to create cache: %v", err)
    }
    defer cacheInstance.Close()
    
    ctx := context.Background()
    
    // 运行基准测试
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        // 在这里实现你的测试逻辑
        for pb.Next() {
            // 例如，随机读写操作
            key := fmt.Sprintf("key:%d", rand.Intn(1000))
            if rand.Intn(100) < 80 {  // 80% 读操作
                cacheInstance.Get(ctx, key)
            } else {  // 20% 写操作
                cacheInstance.Set(ctx, key, []byte("value"), 0)
            }
        }
    })
}
```

### 4.2 实际应用集成测试

你可以参考HTTP服务器示例，将HCache集成到你的应用中：

```go
// 创建缓存实例
cacheInstance, err := cache.NewWithOptions("app-cache",
    cache.WithMaxEntryCount(10000),
    cache.WithShards(32),
    cache.WithTTL(10*time.Minute),
    cache.WithEviction("lfu"),  // 使用LFU策略
)
if err != nil {
    log.Fatalf("Failed to create cache: %v", err)
}
defer cacheInstance.Close()

// 在服务层使用缓存
func (s *Service) GetData(ctx context.Context, id string) (Data, error) {
    cacheKey := fmt.Sprintf("data:%s", id)
    
    // 尝试从缓存获取
    if value, exists, _ := cacheInstance.Get(ctx, cacheKey); exists {
        return value.(Data), nil
    }
    
    // 缓存未命中，从数据库获取
    data, err := s.db.GetData(ctx, id)
    if err != nil {
        return Data{}, err
    }
    
    // 存入缓存
    cacheInstance.Set(ctx, cacheKey, data, 0)
    return data, nil
}
```

## 5. 总结

HCache提供了丰富的示例，帮助用户了解如何在不同场景下使用缓存，以及评估缓存性能。

- **基准测试**：评估基本操作性能和不同配置的影响
- **压力测试**：评估高并发下的性能和稳定性
- **HTTP服务器**：展示实际Web应用中的缓存集成

通过运行和分析这些示例，你可以：
1. 了解HCache的API使用方法
2. 评估不同配置下的性能表现
3. 学习缓存集成的最佳实践
4. 根据你的应用需求选择合适的缓存策略

我们建议在实际应用中，先运行这些示例以熟悉HCache的行为，然后根据应用特点选择合适的配置参数，并通过监控缓存统计来不断优化缓存效果。 