# HCache 使用说明文档

## 1. 简介

HCache 是一个高性能、功能丰富的 Go 语言内存缓存库，专为多核系统环境下的高并发应用场景设计。它提供了灵活的缓存策略、丰富的配置选项和优秀的性能表现，可以轻松集成到各种 Go 应用程序中。

## 2. 安装

### 2.1 使用 go get 安装

```bash
go get github.com/Humphrey-He/hcache
```

### 2.2 使用 go mod 安装

在你的 `go.mod` 文件中添加以下依赖：

```
require github.com/Humphrey-He/hcache v1.0.0  // 请使用最新版本
```

然后运行：

```bash
go mod tidy
```

## 3. 基础使用

### 3.1 创建缓存实例

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/Humphrey-He/hcache/pkg/cache"
)

func main() {
    // 创建一个基础缓存实例
    c, err := cache.NewWithOptions("userCache",
        cache.WithMaxEntryCount(1000),   // 设置最大条目数
        cache.WithTTL(time.Minute*5),    // 设置默认过期时间
    )
    if err != nil {
        panic(err)
    }
    defer c.Close()  // 程序结束时关闭缓存

    // 创建上下文
    ctx := context.Background()

    // 使用缓存...
}
```

### 3.2 基本操作

#### 3.2.1 设置缓存

```go
// 设置缓存，使用默认TTL
err := cache.Set(ctx, "user:1001", userData, 0)

// 设置缓存，指定TTL
err := cache.Set(ctx, "user:1001", userData, time.Hour)

// 设置缓存，仅当键不存在时
ok, err := cache.SetIfNotExists(ctx, "user:1001", userData, time.Hour)
```

#### 3.2.2 获取缓存

```go
// 获取缓存
value, exists, err := cache.Get(ctx, "user:1001")
if err != nil {
    // 处理错误
} else if exists {
    userData := value.(UserData)  // 类型断言
    fmt.Printf("获取到用户数据: %+v\n", userData)
} else {
    fmt.Println("用户数据不存在")
}
```

#### 3.2.3 删除缓存

```go
// 删除单个键
removed, err := cache.Delete(ctx, "user:1001")

// 删除多个键
removedCount, err := cache.DeleteMany(ctx, []string{"user:1001", "user:1002"})

// 清空所有缓存
err := cache.Clear(ctx)
```

#### 3.2.4 批量操作

```go
// 批量设置
items := map[string]interface{}{
    "user:1001": userData1,
    "user:1002": userData2,
}
err := cache.SetMany(ctx, items, time.Hour)

// 批量获取
keys := []string{"user:1001", "user:1002"}
results, err := cache.GetMany(ctx, keys)
for key, result := range results {
    if result.Exists {
        fmt.Printf("键 %s 的值: %v\n", key, result.Value)
    }
}
```

### 3.3 统计信息

```go
// 获取统计信息
stats, err := cache.Stats(ctx)
if err == nil {
    fmt.Printf("命中次数: %d\n", stats.Hits)
    fmt.Printf("未命中次数: %d\n", stats.Misses)
    fmt.Printf("命中率: %.2f%%\n", stats.HitRatio*100)
    fmt.Printf("条目数: %d\n", stats.EntryCount)
    fmt.Printf("内存使用: %d bytes\n", stats.MemoryBytes)
}
```

## 4. 高级特性

### 4.1 缓存旁路模式 (Cache-Aside)

缓存旁路模式允许你为缓存未命中的情况定义数据加载逻辑：

```go
import (
    "github.com/Humphrey-He/hcache/pkg/cache"
    "github.com/Humphrey-He/hcache/pkg/loader"
)

// 创建一个数据加载器
userLoader := loader.NewFunctionLoader(func(ctx context.Context, key string) (interface{}, error) {
    // 从数据库或其他数据源加载数据
    userID := extractUserIDFromKey(key)
    return fetchUserFromDatabase(userID)
})

// 使用加载器创建缓存
c, err := cache.NewWithOptions("userCache",
    cache.WithMaxEntryCount(10000),
    cache.WithLoader(userLoader),
    cache.WithTTL(time.Hour),
)

// 使用 GetOrLoad 方法获取数据，如缓存未命中则自动从加载器获取
userData, err := c.GetOrLoad(ctx, "user:1001")
```

### 4.2 自定义序列化

```go
import (
    "github.com/Humphrey-He/hcache/pkg/cache"
    "github.com/Humphrey-He/hcache/pkg/codec"
)

// 使用 JSON 编解码器
jsonCodec := codec.NewJSONCodec()

// 使用 Gob 编解码器
gobCodec := codec.NewGobCodec()

// 自定义编解码器
c, err := cache.NewWithOptions("myCache",
    cache.WithCodec(jsonCodec),
)
```

### 4.3 不同的淘汰策略

HCache 支持多种淘汰策略：

```go
// LRU (Least Recently Used) - 最近最少使用
c, err := cache.NewWithOptions("myCache", cache.WithEviction("lru"))

// LFU (Least Frequently Used) - 最不经常使用
c, err := cache.NewWithOptions("myCache", cache.WithEviction("lfu"))

// FIFO (First In First Out) - 先进先出
c, err := cache.NewWithOptions("myCache", cache.WithEviction("fifo"))

// Random - 随机淘汰
c, err := cache.NewWithOptions("myCache", cache.WithEviction("random"))
```

### 4.4 准入策略

准入策略用于控制哪些项可以进入缓存，防止缓存抖动：

```go
import (
    "github.com/Humphrey-He/hcache/pkg/admission"
    "github.com/Humphrey-He/hcache/pkg/cache"
)

// 使用 TinyLFU 准入策略
tinyLFU := admission.NewTinyLFU(10000)

c, err := cache.NewWithOptions("myCache",
    cache.WithAdmissionPolicy(tinyLFU),
)
```

## 5. 性能优化

### 5.1 分片设置

增加分片数可以减少锁竞争，提高并发性能：

```go
c, err := cache.NewWithOptions("myCache",
    cache.WithShards(256),  // 默认为 16
)
```

### 5.2 内存限制

设置内存限制可以防止缓存过度消耗系统资源：

```go
c, err := cache.NewWithOptions("myCache",
    cache.WithMaxMemoryBytes(500*1024*1024),  // 限制为 500MB
)
```

### 5.3 性能测试

你可以运行内置的基准测试来评估在你的系统上的性能表现：

```bash
# Linux/macOS
./test/run_benchmarks.sh

# Windows
./test/run_benchmarks.ps1
```

## 6. 错误处理

HCache 提供了标准化的错误类型，便于错误处理：

```go
import "github.com/Humphrey-He/hcache/pkg/errors"

value, exists, err := cache.Get(ctx, "key")
if err != nil {
    switch {
    case errors.Is(err, errors.ErrCacheClosed):
        // 缓存已关闭
    case errors.Is(err, errors.ErrKeyInvalid):
        // 无效的键
    case errors.Is(err, errors.ErrValueTooLarge):
        // 值太大
    default:
        // 其他错误
    }
}
```

## 7. 实际应用场景

### 7.1 HTTP API 响应缓存

```go
func getUserHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := r.URL.Query().Get("id")
    cacheKey := "user:" + userID
    
    // 尝试从缓存获取
    userData, exists, err := userCache.Get(ctx, cacheKey)
    if err != nil {
        http.Error(w, "缓存错误", http.StatusInternalServerError)
        return
    }
    
    if !exists {
        // 从数据库获取
        userData, err = db.GetUser(ctx, userID)
        if err != nil {
            http.Error(w, "数据库错误", http.StatusInternalServerError)
            return
        }
        
        // 存入缓存
        err = userCache.Set(ctx, cacheKey, userData, time.Minute*5)
        if err != nil {
            log.Printf("缓存设置失败: %v", err)
        }
    }
    
    // 返回数据
    json.NewEncoder(w).Encode(userData)
}
```

### 7.2 分布式锁

```go
func acquireLock(ctx context.Context, cache *cache.Cache, lockKey string, ttl time.Duration) (bool, error) {
    return cache.SetIfNotExists(ctx, lockKey, "locked", ttl)
}

func releaseLock(ctx context.Context, cache *cache.Cache, lockKey string) (bool, error) {
    return cache.Delete(ctx, lockKey)
}

// 使用示例
func doWithLock(ctx context.Context, cache *cache.Cache, resourceID string) error {
    lockKey := "lock:" + resourceID
    acquired, err := acquireLock(ctx, cache, lockKey, time.Second*30)
    if err != nil {
        return err
    }
    
    if !acquired {
        return errors.New("无法获取锁")
    }
    
    defer releaseLock(ctx, cache, lockKey)
    
    // 执行需要锁保护的操作...
    
    return nil
}
```

## 8. 故障排除

### 8.1 常见问题

1. **缓存未正确关闭**
   - 确保在应用程序结束时调用 `cache.Close()`
   - 使用 `defer cache.Close()` 确保即使发生异常也能关闭缓存

2. **内存使用过高**
   - 检查 `WithMaxEntryCount` 和 `WithMaxMemoryBytes` 设置
   - 考虑调整 TTL 值

3. **性能下降**
   - 增加分片数 `WithShards` 减少锁竞争
   - 检查缓存命中率，考虑调整缓存大小或淘汰策略

### 8.2 诊断指标

启用指标收集进行性能诊断：

```go
c, err := cache.NewWithOptions("myCache",
    cache.WithMetricsEnabled(true),
)

// 定期输出指标
go func() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        stats, _ := c.Stats(context.Background())
        log.Printf("缓存状态: 条目=%d, 命中率=%.2f%%, 内存=%d bytes",
                  stats.EntryCount, stats.HitRatio*100, stats.MemoryBytes)
    }
}()
```

## 9. 参考链接

- [GitHub 仓库](https://github.com/Humphrey-He/hcache)
- [GoDoc 文档](https://pkg.go.dev/github.com/Humphrey-He/hcache)
- [示例代码](https://github.com/Humphrey-He/hcache/tree/main/examples) 