# HCache

<div align="center">

[![Go Reference](https://pkg.go.dev/badge/github.com/Humphrey-He/hcache.svg)](https://pkg.go.dev/github.com/Humphrey-He/hcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/Humphrey-He/hcache)](https://goreportcard.com/report/github.com/Humphrey-He/hcache)
[![License](https://img.shields.io/github/license/Humphrey-He/hcache)](LICENSE)
[![Build Status](https://github.com/Humphrey-He/hcache/workflows/build/badge.svg)](https://github.com/Humphrey-He/hcache/actions)
[![Coverage](https://codecov.io/gh/Humphrey-He/hcache/branch/main/graph/badge.svg)](https://codecov.io/gh/Humphrey-He/hcache)

<p>一个高性能、功能丰富的 Go 语言内存缓存库</p>
</div>

## 📋 目录

- [特性](#-特性)
- [安装](#-安装)
- [快速开始](#-快速开始)
- [使用示例](#-使用示例)
  - [基本操作](#基本操作)
  - [缓存旁路模式](#缓存旁路模式)
  - [HTTP 服务器集成](#http-服务器集成)
- [架构](#-架构)
- [性能](#-性能)
- [配置](#-配置)
- [高级功能](#-高级功能)
- [贡献](#-贡献)
- [许可证](#-许可证)

## ✨ 特性

- **高性能**：针对多核系统优化，分片设计最小化锁竞争
- **灵活的淘汰策略**：支持 LRU、LFU、FIFO 和随机淘汰策略
- **TTL 支持**：自动过期缓存条目，支持自定义存活时间
- **指标收集**：详细的性能指标，用于监控缓存效率
- **准入控制**：通过智能准入策略防止缓存抖动
- **并发安全**：线程安全操作，适用于并发环境
- **内存限制**：可配置内存限制，防止内存溢出
- **序列化支持**：可插拔编解码器，支持值序列化和压缩
- **数据加载**：内置支持数据加载器和缓存旁路模式
- **可扩展**：模块化设计允许核心组件的自定义实现

## 📦 安装

```bash
go get github.com/Humphrey-He/hcache
```

## 🚀 快速开始

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Humphrey-He/hcache/pkg/cache"
)

func main() {
	// 创建一个使用默认配置的缓存
	c, err := cache.NewWithOptions("myCache",
		cache.WithMaxEntryCount(1000),
		cache.WithTTL(time.Minute*5),
	)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	ctx := context.Background()

	// 设置一个值
	err = c.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		panic(err)
	}

	// 获取一个值
	value, exists, err := c.Get(ctx, "key1")
	if err != nil {
		panic(err)
	}

	if exists {
		fmt.Printf("值: %v\n", value)
	} else {
		fmt.Println("未找到键")
	}
}
```

## 📝 使用示例

### 基本操作

```go
// 设置带 TTL 的值
cache.Set(ctx, "user:1001", userData, time.Hour)

// 获取值
value, exists, err := cache.Get(ctx, "user:1001")

// 删除值
removed, err := cache.Delete(ctx, "user:1001")

// 清除所有条目
cache.Clear(ctx)

// 获取统计信息
stats, err := cache.Stats(ctx)
fmt.Printf("命中: %d, 未命中: %d, 命中率: %.2f%%\n", 
           stats.Hits, stats.Misses, stats.HitRatio*100)
```

### 缓存旁路模式

```go
import "github.com/Humphrey-He/hcache/pkg/loader"

// 创建数据加载器
userLoader := loader.NewFunctionLoader(func(ctx context.Context, key string) (interface{}, error) {
    // 当缓存中不存在时从数据库获取数据
    return fetchUserFromDatabase(key)
})

// 创建带加载器的缓存
c, err := cache.NewWithOptions("userCache",
    cache.WithMaxEntryCount(10000),
    cache.WithLoader(userLoader),
    cache.WithTTL(time.Hour),
)

// 获取或加载数据
userData, err := c.GetOrLoad(ctx, "user:1001")
```

### HTTP 服务器集成

查看 [examples/http_server](examples/http_server) 目录获取完整的 HCache 与 HTTP 服务器集成示例。

## 🏗️ 架构

HCache 采用分层架构设计，注重性能、灵活性和可扩展性：

- **pkg/**: 公共 API 和接口，供外部使用
  - **cache/**: 主要缓存接口和实现
  - **loader/**: 缓存未命中时的数据加载接口
  - **codec/**: 序列化接口和实现
  - **errors/**: 标准化错误类型
  
- **internal/**: 实现细节（不供外部使用）
  - **metrics/**: 性能指标收集
  - **storage/**: 内部数据存储机制
  - **eviction/**: 淘汰策略实现
  - **ttl/**: 存活时间管理
  - **admission/**: 准入策略实现
  - **utils/**: 实用函数和数据结构

## 📊 性能

HCache 专为多核环境下的高性能设计。基准测试套件涵盖各种场景，包括不同的：

- 缓存大小
- 并发级别
- 访问模式（包括 Zipfian 分布）
- 值大小
- 读写比例
- 淘汰策略

运行基准测试：

```bash
# Linux/macOS
./test/run_benchmarks.sh

# Windows
./test/run_benchmarks.ps1
```

基准测试结果示例（Intel Core i7，16GB RAM）：

| 基准测试 | 操作次数 | ns/op | B/op | allocs/op |
|---------|---------|-------|------|-----------|
| Get/Hit | 20000000 | 63.1 | 8 | 1 |
| Get/Miss | 10000000 | 115.0 | 24 | 2 |
| Set/New | 5000000 | 235.0 | 40 | 3 |
| Set/Existing | 5000000 | 210.0 | 32 | 2 |
| Mixed/Read80Write20 | 5000000 | 180.0 | 32 | 2 |

## ⚙️ 配置

HCache 提供灵活的配置系统，使用函数选项模式：

```go
cache, err := cache.NewWithOptions("myCache",
    // 基本设置
    cache.WithMaxEntryCount(100000),          // 最大条目数
    cache.WithMaxMemoryBytes(500*1024*1024),  // 内存限制（500MB）
    cache.WithShards(256),                    // 分片数量，用于并发
    
    // 淘汰设置
    cache.WithEviction("lru"),                // LRU、LFU、FIFO 或 Random
    cache.WithTTL(time.Hour),                 // 默认 TTL
    
    // 高级设置
    cache.WithMetricsEnabled(true),           // 启用指标收集
    cache.WithAdmissionPolicy(myPolicy),      // 自定义准入策略
    cache.WithCodec(myCodec),                 // 自定义序列化
    cache.WithLoader(myLoader),               // 缓存未命中时的数据加载器
)
```

## 🔧 高级功能

### 自定义序列化

```go
import "github.com/Humphrey-He/hcache/pkg/codec"

// 创建自定义编解码器
myCodec := codec.NewJSONCodec()

// 在缓存中使用编解码器
c, err := cache.NewWithOptions("myCache",
    cache.WithCodec(myCodec),
)
```

### 自定义准入策略

```go
import "github.com/Humphrey-He/hcache/pkg/admission"

// 创建自定义准入策略
myPolicy := admission.NewTinyLFU(10000)

// 在缓存中使用准入策略
c, err := cache.NewWithOptions("myCache",
    cache.WithAdmissionPolicy(myPolicy),
)
```

### 指标收集

```go
// 启用指标
c, err := cache.NewWithOptions("myCache",
    cache.WithMetricsEnabled(true),
    cache.WithMetricsLevel("detailed"),  // "basic"、"detailed" 或 "disabled"
)

// 获取基本统计信息
stats, _ := c.Stats(ctx)
fmt.Printf("条目数: %d\n", stats.EntryCount)
fmt.Printf("命中率: %.2f%%\n", float64(stats.Hits)/(float64(stats.Hits+stats.Misses))*100)
fmt.Printf("内存使用: %.2f MB\n", float64(stats.Size)/(1024*1024))
fmt.Printf("淘汰次数: %d\n", stats.Evictions)

// 启用详细指标后，可获取更多信息
detailedStats := stats.(*cache.DetailedStats)  // 类型断言获取详细统计信息
fmt.Printf("平均获取延迟: %v\n", detailedStats.AvgGetLatency)
fmt.Printf("P99获取延迟: %v\n", detailedStats.P99GetLatency)
fmt.Printf("缓存碎片率: %.2f%%\n", detailedStats.FragmentationRatio*100)
```

HCache 还可以将指标导出到 Prometheus：

```go
import "github.com/Humphrey-He/hcache/pkg/metrics"

// 注册缓存指标到 Prometheus
metrics.RegisterPrometheus(c, "myapp_cache")
```

## 🔌 并发应用示例

HCache 专为高并发场景设计，以下是一个并发应用示例：

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/Humphrey-He/hcache/pkg/cache"
)

func main() {
    // 创建具有多分片的高并发缓存
    c, _ := cache.NewWithOptions("concurrentCache",
        cache.WithMaxEntryCount(100000),
        cache.WithShards(256),  // 256个分片以最小化锁竞争
    )
    
    ctx := context.Background()
    var wg sync.WaitGroup
    
    // 模拟100个并发协程
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            // 每个协程执行1000次操作
            for j := 0; j < 1000; j++ {
                key := fmt.Sprintf("key:%d:%d", id, j)
                
                // 80%读操作，20%写操作
                if j%5 == 0 {
                    c.Set(ctx, key, fmt.Sprintf("value:%d:%d", id, j), time.Minute)
                } else {
                    c.Get(ctx, key)
                }
            }
        }(i)
    }
    
    wg.Wait()
    stats, _ := c.Stats(ctx)
    fmt.Printf("完成100,000次并发操作\n")
    fmt.Printf("命中率: %.2f%%\n", float64(stats.Hits)*100/float64(stats.Hits+stats.Misses))
}
```

## 📊 性能基准测试

### 测试环境

所有基准测试在以下环境中进行：

- **CPU**: AMD Ryzen 5 5600G with Radeon Graphics
- **内存**: 16GB DDR4-3200
- **操作系统**: Windows 10
- **Go 版本**: 1.18+
- **测试时长**: 每个基准测试重复3次，每次运行3秒

### 核心操作性能

| 操作 | 缓存大小 | 性能 (ns/op) | 内存 (B/op) | 分配次数 (allocs/op) |
|-----------|------------|---------------------|---------------|-------------------------|
| Get/命中 | 1,000 | 97.47 | 0 | 0 |
| Get/命中 | 10,000 | 97.31 | 0 | 0 |
| Get/命中 | 100,000 | 98.98 | 0 | 0 |
| Get/未命中 | 1,000 | 128.33 | 24 | 2 |
| Get/未命中 | 10,000 | 129.30 | 24 | 1 |
| Get/未命中 | 100,000 | 123.87 | 24 | 1 |
| Set/新增 | 1,000 | 439.13 | 72.7 | 3 |
| Set/新增 | 10,000 | 442.03 | 72.3 | 3 |
| Set/新增 | 100,000 | 456.80 | 70.7 | 3 |
| Set/更新 | 1,000 | 179.03 | 24 | 1 |
| Set/更新 | 10,000 | 162.60 | 24 | 1 |
| Set/更新 | 100,000 | 170.93 | 24 | 1 |

**分析**:
- **Get/命中性能**: 极其高效，约97-99ns，零内存分配
- **Get/未命中性能**: 仍然很快，约123-129ns，最小化内存分配
- **Set操作**: 新条目约440-457ns，更新现有条目更快，约163-179ns
- **缓存大小影响**: 性能在不同缓存大小下保持稳定，100,000条目时仅有轻微降低
- **内存效率**: 读操作（Get/命中）零内存分配，展示出极佳的效率

### 淘汰策略比较

| 淘汰策略 | 性能 (ns/op) | 内存 (B/op) | 分配次数 (allocs/op) | 排名 |
|-----------------|---------------------|---------------|-------------------------|---------|
| LRU | 123.60 | 12 | 0 | 4 |
| LFU | 119.03 | 12 | 0 | 3 |
| FIFO | 119.00 | 12 | 0 | 1 |
| Random | 118.83 | 12 | 0 | 2 |

**分析**:
- 所有淘汰策略性能相近，差异在5%以内
- Random和FIFO由于决策逻辑简单，略快一些
- LRU由于需要追踪最近访问时间，开销略高
- 所有策略都展现出优秀的内存效率，分配模式一致
- LFU的内存使用得到很好优化，与简单策略相比没有额外开销

### 并发性能

在不同并发级别和读写比例下进行的压力测试：

| 场景 | 线程数 | QPS | 读写比 | 成功率 | 最大延迟 | 命中率 |
|----------|---------|-----|------------|--------------|-------------|----------|
| 标准负载 | 4 | 1000 | 80%/20% | 100% | 1.00ms | 24.38% |
| 高并发 | 8 | 2000 | 80%/20% | 100% | 0.48ms | 37.46% |
| 写密集 | 8 | 2000 | 20%/80% | 100% | 0.44ms | 77.43% |

**分析**:
- HCache在高负载下保持100%成功率
- 所有场景下延迟都保持在亚毫秒级
- 更高并发反而显示更低的最大延迟，证明分片策略有效
- 写密集工作负载由于缓存填充更快，达到更高命中率
- 缓存性能随QPS和线程数线性扩展

## 👥 贡献

欢迎贡献！请随时提交 Pull Request。对于重大更改，请先开 issue 讨论您想要更改的内容。

1. Fork 仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

请确保适当更新测试。

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。 