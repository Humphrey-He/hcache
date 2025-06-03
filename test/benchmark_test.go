// Package test provides comprehensive benchmark tests for the HCache library.
// These benchmarks measure performance characteristics under various workloads,
// configurations, and access patterns to ensure optimal cache performance.
//
// Package test 为HCache库提供全面的基准测试。
// 这些基准测试在各种工作负载、配置和访问模式下测量性能特征，以确保最佳的缓存性能。
package test

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/Humphrey-He/hcache/pkg/cache"
)

// BenchmarkCacheOperations benchmarks basic cache operations.
// It tests various combinations of cache sizes and shard counts to measure
// how these parameters affect performance under different workloads.
//
// Parameters:
//   - b: The benchmark context
//
// BenchmarkCacheOperations 对基本缓存操作进行基准测试。
// 它测试缓存大小和分片数量的各种组合，以测量这些参数如何在不同工作负载下影响性能。
//
// 参数:
//   - b: 基准测试上下文
func BenchmarkCacheOperations(b *testing.B) {
	// Disable GC during benchmarks to reduce noise
	// 在基准测试期间禁用GC以减少干扰
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	// Run benchmarks with different cache configurations
	// 使用不同的缓存配置运行基准测试
	cacheSizes := []int{10000, 100000, 1000000}
	shardCounts := []int{16, 64, 256}

	for _, size := range cacheSizes {
		for _, shards := range shardCounts {
			name := fmt.Sprintf("Size=%d/Shards=%d", size, shards)
			b.Run(name, func(b *testing.B) {
				runCacheBenchmarks(b, size, shards)
			})
		}
	}
}

// runCacheBenchmarks runs all benchmarks for a specific cache configuration.
// It creates a cache with the given size and shard count, then runs various
// benchmark scenarios to test different aspects of cache performance.
//
// Parameters:
//   - b: The benchmark context
//   - cacheSize: The maximum number of entries the cache can hold
//   - shardCount: The number of shards to divide the cache into
//
// runCacheBenchmarks 为特定缓存配置运行所有基准测试。
// 它创建具有给定大小和分片数量的缓存，然后运行各种基准测试场景以测试缓存性能的不同方面。
//
// 参数:
//   - b: 基准测试上下文
//   - cacheSize: 缓存可以容纳的最大条目数
//   - shardCount: 将缓存分成的分片数量
func runCacheBenchmarks(b *testing.B, cacheSize, shardCount int) {
	// Create cache with the specified configuration
	// 使用指定的配置创建缓存
	cacheInstance, err := cache.NewWithOptions("benchmark-cache",
		cache.WithMaxEntryCount(cacheSize),
		cache.WithShards(shardCount),
		cache.WithTTL(time.Hour),
		cache.WithMetricsEnabled(false), // Disable metrics for benchmarking
	)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheInstance.Close()

	ctx := context.Background()

	// Generate test data
	// 生成测试数据
	keys := make([]string, cacheSize)
	values := make([][]byte, cacheSize)
	for i := 0; i < cacheSize; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
		values[i] = make([]byte, 1024)
		rand.Read(values[i])
	}

	// Run individual benchmarks
	// 运行单独的基准测试
	b.Run("Get/Hit", func(b *testing.B) {
		benchmarkGetHit(b, ctx, cacheInstance, keys, values)
	})

	b.Run("Get/Miss", func(b *testing.B) {
		benchmarkGetMiss(b, ctx, cacheInstance, keys)
	})

	b.Run("Set/New", func(b *testing.B) {
		benchmarkSetNew(b, ctx, cacheInstance, keys, values)
	})

	b.Run("Set/Existing", func(b *testing.B) {
		benchmarkSetExisting(b, ctx, cacheInstance, keys, values)
	})

	b.Run("Delete", func(b *testing.B) {
		benchmarkDelete(b, ctx, cacheInstance, keys)
	})

	b.Run("Mixed/Read80Write20", func(b *testing.B) {
		benchmarkMixed(b, ctx, cacheInstance, keys, values, 80)
	})

	b.Run("Mixed/Read50Write50", func(b *testing.B) {
		benchmarkMixed(b, ctx, cacheInstance, keys, values, 50)
	})

	b.Run("Mixed/Read20Write80", func(b *testing.B) {
		benchmarkMixed(b, ctx, cacheInstance, keys, values, 20)
	})

	b.Run("ZipfianAccess", func(b *testing.B) {
		benchmarkZipfianAccess(b, ctx, cacheInstance, keys, values)
	})
}

// benchmarkGetHit benchmarks cache hit performance.
// It preloads the cache with data and then measures the performance
// of retrieving keys that are known to be in the cache.
//
// Parameters:
//   - b: The benchmark context
//   - ctx: The context for cache operations
//   - cacheInstance: The cache instance to benchmark
//   - keys: The keys to use for the benchmark
//   - values: The values to use for the benchmark
//
// benchmarkGetHit 对缓存命中性能进行基准测试。
// 它预先加载缓存数据，然后测量检索已知在缓存中的键的性能。
//
// 参数:
//   - b: 基准测试上下文
//   - ctx: 缓存操作的上下文
//   - cacheInstance: 要测试的缓存实例
//   - keys: 用于基准测试的键
//   - values: 用于基准测试的值
func benchmarkGetHit(b *testing.B, ctx context.Context, cacheInstance cache.ICache, keys []string, values [][]byte) {
	// Preload cache
	// 预加载缓存
	for i := 0; i < 1000 && i < len(keys); i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Access keys that are definitely in the cache
			// 访问肯定在缓存中的键
			key := keys[i%1000]
			_, exists, err := cacheInstance.Get(ctx, key)
			if err != nil {
				b.Fatalf("Failed to get from cache: %v", err)
			}
			if !exists {
				b.Fatalf("Expected cache hit for key %s", key)
			}
			i++
		}
	})
}

// benchmarkGetMiss benchmarks cache miss performance.
// It measures the performance of attempting to retrieve keys
// that are known not to be in the cache.
//
// Parameters:
//   - b: The benchmark context
//   - ctx: The context for cache operations
//   - cacheInstance: The cache instance to benchmark
//   - keys: The keys to use for the benchmark (not actually used)
//
// benchmarkGetMiss 对缓存未命中性能进行基准测试。
// 它测量尝试检索已知不在缓存中的键的性能。
//
// 参数:
//   - b: 基准测试上下文
//   - ctx: 缓存操作的上下文
//   - cacheInstance: 要测试的缓存实例
//   - keys: 用于基准测试的键（实际上未使用）
func benchmarkGetMiss(b *testing.B, ctx context.Context, cacheInstance cache.ICache, keys []string) {
	// Clear the cache first
	// 首先清除缓存
	cacheInstance.Clear(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Use keys that shouldn't be in the cache
			// 使用不应该在缓存中的键
			key := fmt.Sprintf("missing:%d", i)
			_, exists, err := cacheInstance.Get(ctx, key)
			if err != nil {
				b.Fatalf("Failed to get from cache: %v", err)
			}
			if exists {
				b.Fatalf("Unexpected cache hit for key %s", key)
			}
			i++
		}
	})
}

// benchmarkSetNew benchmarks the performance of adding new keys to the cache.
// It clears the cache first to ensure all operations are adding new entries.
//
// Parameters:
//   - b: The benchmark context
//   - ctx: The context for cache operations
//   - cacheInstance: The cache instance to benchmark
//   - keys: The keys to use for the benchmark
//   - values: The values to use for the benchmark
//
// benchmarkSetNew 对向缓存添加新键的性能进行基准测试。
// 它首先清除缓存以确保所有操作都是添加新条目。
//
// 参数:
//   - b: 基准测试上下文
//   - ctx: 缓存操作的上下文
//   - cacheInstance: 要测试的缓存实例
//   - keys: 用于基准测试的键
//   - values: 用于基准测试的值
func benchmarkSetNew(b *testing.B, ctx context.Context, cacheInstance cache.ICache, keys []string, values [][]byte) {
	// Clear the cache first
	// 首先清除缓存
	cacheInstance.Clear(ctx)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("new:%d", i)
			err := cacheInstance.Set(ctx, key, values[i%len(values)], time.Hour)
			if err != nil {
				b.Fatalf("Failed to set cache: %v", err)
			}
			i++
		}
	})
}

// benchmarkSetExisting benchmarks the performance of updating existing keys.
// It preloads the cache with data and then measures the performance
// of updating those same keys with new values.
//
// Parameters:
//   - b: The benchmark context
//   - ctx: The context for cache operations
//   - cacheInstance: The cache instance to benchmark
//   - keys: The keys to use for the benchmark
//   - values: The values to use for the benchmark
//
// benchmarkSetExisting 对更新现有键的性能进行基准测试。
// 它预先加载缓存数据，然后测量用新值更新这些相同键的性能。
//
// 参数:
//   - b: 基准测试上下文
//   - ctx: 缓存操作的上下文
//   - cacheInstance: 要测试的缓存实例
//   - keys: 用于基准测试的键
//   - values: 用于基准测试的值
func benchmarkSetExisting(b *testing.B, ctx context.Context, cacheInstance cache.ICache, keys []string, values [][]byte) {
	// Preload cache
	// 预加载缓存
	for i := 0; i < 1000 && i < len(keys); i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Update keys that are definitely in the cache
			// 更新肯定在缓存中的键
			key := keys[i%1000]
			err := cacheInstance.Set(ctx, key, values[(i+1)%len(values)], time.Hour)
			if err != nil {
				b.Fatalf("Failed to set cache: %v", err)
			}
			i++
		}
	})
}

// benchmarkDelete benchmarks the performance of deleting keys from the cache.
// It preloads the cache with enough keys for the benchmark and then
// measures the performance of deleting those keys.
//
// Parameters:
//   - b: The benchmark context
//   - ctx: The context for cache operations
//   - cacheInstance: The cache instance to benchmark
//   - keys: The keys to use for the benchmark
//
// benchmarkDelete 对从缓存中删除键的性能进行基准测试。
// 它为基准测试预加载足够的键，然后测量删除这些键的性能。
//
// 参数:
//   - b: 基准测试上下文
//   - ctx: 缓存操作的上下文
//   - cacheInstance: 要测试的缓存实例
//   - keys: 用于基准测试的键
func benchmarkDelete(b *testing.B, ctx context.Context, cacheInstance cache.ICache, keys []string) {
	// Preload cache with enough keys for the benchmark
	// 为基准测试预加载足够的键
	for i := 0; i < b.N && i < len(keys); i++ {
		key := fmt.Sprintf("delete:%d", i)
		err := cacheInstance.Set(ctx, key, []byte("value"), time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("delete:%d", i)
			_, err := cacheInstance.Delete(ctx, key)
			if err != nil {
				b.Fatalf("Failed to delete from cache: %v", err)
			}
			i++
		}
	})
}

// benchmarkMixed benchmarks a mix of get and set operations.
// It simulates a realistic workload with a specified ratio of
// read operations to write operations.
//
// Parameters:
//   - b: The benchmark context
//   - ctx: The context for cache operations
//   - cacheInstance: The cache instance to benchmark
//   - keys: The keys to use for the benchmark
//   - values: The values to use for the benchmark
//   - readPercentage: The percentage of operations that should be reads
//
// benchmarkMixed 对混合的获取和设置操作进行基准测试。
// 它模拟具有指定读取操作与写入操作比率的真实工作负载。
//
// 参数:
//   - b: 基准测试上下文
//   - ctx: 缓存操作的上下文
//   - cacheInstance: 要测试的缓存实例
//   - keys: 用于基准测试的键
//   - values: 用于基准测试的值
//   - readPercentage: 应该是读取操作的百分比
func benchmarkMixed(b *testing.B, ctx context.Context, cacheInstance cache.ICache, keys []string, values [][]byte, readPercentage int) {
	// Preload cache
	// 预加载缓存
	for i := 0; i < 1000 && i < len(keys); i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Decide whether to do a read or write operation
			// 决定是执行读取还是写入操作
			isRead := rand.Intn(100) < readPercentage

			// Choose a random key
			// 选择一个随机键
			key := keys[i%len(keys)]

			if isRead {
				// Read operation
				// 读操作
				_, _, err := cacheInstance.Get(ctx, key)
				if err != nil {
					b.Fatalf("Failed to get from cache: %v", err)
				}
			} else {
				// Write operation
				// 写操作
				err := cacheInstance.Set(ctx, key, values[i%len(values)], time.Hour)
				if err != nil {
					b.Fatalf("Failed to set cache: %v", err)
				}
			}
			i++
		}
	})
}

// benchmarkZipfianAccess benchmarks cache performance with zipfian access patterns.
// It simulates real-world access patterns where some keys are accessed much more
// frequently than others, following a power law distribution.
//
// Parameters:
//   - b: The benchmark context
//   - ctx: The context for cache operations
//   - cacheInstance: The cache instance to benchmark
//   - keys: The keys to use for the benchmark
//   - values: The values to use for the benchmark
//
// benchmarkZipfianAccess 对缓存在齐普夫访问模式下的性能进行基准测试。
// 它模拟现实世界的访问模式，其中一些键比其他键访问频率高得多，遵循幂律分布。
//
// 参数:
//   - b: 基准测试上下文
//   - ctx: 缓存操作的上下文
//   - cacheInstance: 要测试的缓存实例
//   - keys: 用于基准测试的键
//   - values: 用于基准测试的值
func benchmarkZipfianAccess(b *testing.B, ctx context.Context, cacheInstance cache.ICache, keys []string, values [][]byte) {
	// Preload cache
	// 预加载缓存
	for i := 0; i < 1000 && i < len(keys); i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		// 在每个并发goroutine中创建独立的随机数生成器和Zipf对象
		// Create separate random number generator and Zipf object for each goroutine
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		keyCount := uint64(len(keys))
		if keyCount <= 1 {
			b.Fatal("Not enough keys for zipfian access test")
		}
		zipf := rand.NewZipf(localRand, 1.5, 1.0, keyCount-1)

		for pb.Next() {
			// 获取齐普夫分布的索引，并确保在有效范围内
			// Get a zipfian distributed index and ensure it's in valid range
			idx := zipf.Uint64() % keyCount
			if idx >= keyCount {
				idx = keyCount - 1
			}

			key := keys[idx]

			// 80% reads, 20% writes
			// 80%读取，20%写入
			if localRand.Intn(100) < 80 {
				_, _, err := cacheInstance.Get(ctx, key)
				if err != nil {
					b.Fatalf("Failed to get from cache: %v", err)
				}
			} else {
				valIdx := idx
				if valIdx >= uint64(len(values)) {
					valIdx = uint64(len(values)) - 1
				}
				err := cacheInstance.Set(ctx, key, values[valIdx], time.Hour)
				if err != nil {
					b.Fatalf("Failed to set cache: %v", err)
				}
			}
		}
	})
}

// BenchmarkConcurrency benchmarks cache performance under different levels of concurrency.
// It tests how the cache performs as the number of concurrent goroutines increases,
// which helps identify potential bottlenecks in high-concurrency environments.
//
// Parameters:
//   - b: The benchmark context
//
// BenchmarkConcurrency 在不同并发级别下对缓存性能进行基准测试。
// 它测试随着并发goroutine数量的增加，缓存的性能表现如何，
// 这有助于识别高并发环境中的潜在瓶颈。
//
// 参数:
//   - b: 基准测试上下文
func BenchmarkConcurrency(b *testing.B) {
	concurrencyLevels := []int{1, 4, 16, 64, 256}

	for _, concurrency := range concurrencyLevels {
		name := fmt.Sprintf("Concurrency=%d", concurrency)
		b.Run(name, func(b *testing.B) {
			benchmarkWithConcurrency(b, concurrency)
		})
	}
}

// benchmarkWithConcurrency benchmarks cache performance with a specific concurrency level.
// It creates a fixed number of goroutines that all access the cache simultaneously,
// simulating a specific level of concurrent access.
//
// Parameters:
//   - b: The benchmark context
//   - concurrency: The number of concurrent goroutines to use
//
// benchmarkWithConcurrency 使用特定并发级别对缓存性能进行基准测试。
// 它创建固定数量的goroutine，这些goroutine同时访问缓存，
// 模拟特定级别的并发访问。
//
// 参数:
//   - b: 基准测试上下文
//   - concurrency: 要使用的并发goroutine数量
func benchmarkWithConcurrency(b *testing.B, concurrency int) {
	// Create cache
	// 创建缓存
	cacheInstance, err := cache.NewWithOptions("concurrency-cache",
		cache.WithMaxEntryCount(100000),
		cache.WithShards(16),
		cache.WithTTL(time.Hour),
		cache.WithMetricsEnabled(false),
	)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheInstance.Close()

	ctx := context.Background()

	// Generate test data
	// 生成测试数据
	const keyCount = 10000
	keys := make([]string, keyCount)
	values := make([][]byte, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
		values[i] = make([]byte, 128)
		rand.Read(values[i])
	}

	// Preload cache
	// 预加载缓存
	for i := 0; i < keyCount; i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Create a wait group to synchronize goroutines
	// 创建一个等待组以同步goroutine
	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Calculate operations per goroutine
	// 计算每个goroutine的操作数
	opsPerGoroutine := b.N / concurrency
	if opsPerGoroutine < 1 {
		opsPerGoroutine = 1
	}

	// Start goroutines
	// 启动goroutine
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()

			// Perform mixed operations
			// 执行混合操作
			for j := 0; j < opsPerGoroutine; j++ {
				// 80% reads, 20% writes
				// 80%读取，20%写入
				isRead := rand.Intn(100) < 80

				// Choose a random key
				// 选择一个随机键
				keyIdx := rand.Intn(keyCount)
				key := keys[keyIdx]

				if isRead {
					_, _, err := cacheInstance.Get(ctx, key)
					if err != nil {
						b.Errorf("Failed to get from cache: %v", err)
						return
					}
				} else {
					err := cacheInstance.Set(ctx, key, values[keyIdx], time.Hour)
					if err != nil {
						b.Errorf("Failed to set cache: %v", err)
						return
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	// 等待所有goroutine完成
	wg.Wait()
}

// BenchmarkEvictionPolicies benchmarks different eviction policies.
// It compares the performance of LRU, LFU, FIFO, and random eviction
// strategies under the same workload to help choose the optimal policy.
//
// Parameters:
//   - b: The benchmark context
//
// BenchmarkEvictionPolicies 对不同的淘汰策略进行基准测试。
// 它比较了LRU、LFU、FIFO和随机淘汰策略在相同工作负载下的性能，
// 以帮助选择最佳策略。
//
// 参数:
//   - b: 基准测试上下文
func BenchmarkEvictionPolicies(b *testing.B) {
	evictionPolicies := []string{"lru", "lfu", "fifo", "random"}

	for _, policy := range evictionPolicies {
		name := fmt.Sprintf("Policy=%s", policy)
		b.Run(name, func(b *testing.B) {
			benchmarkEvictionPolicy(b, policy)
		})
	}
}

// benchmarkEvictionPolicy benchmarks a specific eviction policy.
// It creates a small cache that will trigger evictions and measures
// the performance impact of the specified eviction algorithm.
//
// Parameters:
//   - b: The benchmark context
//   - policy: The eviction policy to benchmark
//
// benchmarkEvictionPolicy 对特定淘汰策略进行基准测试。
// 它创建一个会触发淘汰的小缓存，并测量指定淘汰算法的性能影响。
//
// 参数:
//   - b: 基准测试上下文
//   - policy: 要测试的淘汰策略
func benchmarkEvictionPolicy(b *testing.B, policy string) {
	// Create a small cache that will trigger evictions
	// 创建一个会触发淘汰的小缓存
	cacheInstance, err := cache.NewWithOptions("eviction-cache",
		cache.WithMaxEntryCount(1000),
		cache.WithShards(16),
		cache.WithTTL(time.Hour),
		cache.WithEviction(policy),
		cache.WithMetricsEnabled(false),
	)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheInstance.Close()

	ctx := context.Background()

	// Generate test data - 10x the cache size to force evictions
	// 生成测试数据 - 缓存大小的10倍以强制淘汰
	keys := make([]string, 10000)
	values := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
		values[i] = make([]byte, 128)
		rand.Read(values[i])
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Alternate between setting and getting
			// 在设置和获取之间交替
			if i%2 == 0 {
				// Set operation - use keys that will cause eviction
				// 设置操作 - 使用会导致淘汰的键
				key := keys[i%len(keys)]
				err := cacheInstance.Set(ctx, key, values[i%len(values)], time.Hour)
				if err != nil {
					// Ignore eviction errors
					// 忽略淘汰错误
					if err.Error() != "cache: cache is full" {
						b.Fatalf("Failed to set cache: %v", err)
					}
				}
			} else {
				// Get operation - use a zipfian distribution to simulate hot keys
				// 获取操作 - 使用齐普夫分布模拟热门键
				idx := i % 1000 // Focus on the first 1000 keys
				key := keys[idx]
				_, _, err := cacheInstance.Get(ctx, key)
				if err != nil {
					b.Fatalf("Failed to get from cache: %v", err)
				}
			}
			i++
		}
	})
}

// BenchmarkTTL benchmarks TTL (Time To Live) expiration performance.
// It tests how the cache performs when items are constantly
// expiring due to short TTLs, which is common in caching scenarios
// where data freshness is critical.
//
// Parameters:
//   - b: The benchmark context
//
// BenchmarkTTL 对TTL（存活时间）过期性能进行基准测试。
// 它测试当项目由于短TTL而不断过期时缓存的性能表现，
// 这在数据新鲜度至关重要的缓存场景中很常见。
//
// 参数:
//   - b: 基准测试上下文
func BenchmarkTTL(b *testing.B) {
	ttlValues := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		10 * time.Second,
	}

	for _, ttl := range ttlValues {
		name := fmt.Sprintf("TTL=%s", ttl)
		b.Run(name, func(b *testing.B) {
			benchmarkWithTTL(b, ttl)
		})
	}
}

// benchmarkWithTTL benchmarks cache performance with a specific TTL value.
// It measures the overhead of TTL checking and the impact of
// expired entries on cache performance.
//
// Parameters:
//   - b: The benchmark context
//   - ttl: The TTL duration to use for cache entries
//
// benchmarkWithTTL 使用特定TTL值对缓存性能进行基准测试。
// 它测量TTL检查的开销以及过期条目对缓存性能的影响。
//
// 参数:
//   - b: 基准测试上下文
//   - ttl: 用于缓存条目的TTL持续时间
func benchmarkWithTTL(b *testing.B, ttl time.Duration) {
	// Create cache with the specified TTL
	// 创建具有指定TTL的缓存
	cacheInstance, err := cache.NewWithOptions("ttl-cache",
		cache.WithMaxEntryCount(10000),
		cache.WithShards(16),
		cache.WithTTL(ttl),
		cache.WithMetricsEnabled(false),
	)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheInstance.Close()

	ctx := context.Background()

	// Generate test data
	// 生成测试数据
	keys := make([]string, 1000)
	values := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
		values[i] = make([]byte, 128)
		rand.Read(values[i])
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Every 100 iterations, set a batch of keys
			// 每100次迭代，设置一批键
			if i%100 == 0 {
				for j := 0; j < 10; j++ {
					key := keys[(i+j)%len(keys)]
					err := cacheInstance.Set(ctx, key, values[(i+j)%len(values)], ttl)
					if err != nil {
						b.Fatalf("Failed to set cache: %v", err)
					}
				}
				// Sleep to allow some keys to expire if TTL is short
				// 休眠以允许一些键过期（如果TTL较短）
				if ttl < time.Second {
					time.Sleep(ttl / 2)
				}
			}

			// Get operation - will sometimes hit expired entries
			// 获取操作 - 有时会命中过期条目
			key := keys[i%len(keys)]
			_, _, err := cacheInstance.Get(ctx, key)
			if err != nil {
				b.Fatalf("Failed to get from cache: %v", err)
			}
			i++
		}
	})
}

// BenchmarkHitRatio benchmarks cache performance with different hit ratios.
// It tests how the cache performs with varying levels of cache hit rates,
// which helps understand performance characteristics under different workloads.
//
// Parameters:
//   - b: The benchmark context
//
// BenchmarkHitRatio 使用不同命中率对缓存性能进行基准测试。
// 它测试缓存在不同缓存命中率水平下的性能表现，
// 这有助于了解不同工作负载下的性能特征。
//
// 参数:
//   - b: 基准测试上下文
func BenchmarkHitRatio(b *testing.B) {
	hitRatios := []int{0, 25, 50, 75, 100}

	for _, ratio := range hitRatios {
		name := fmt.Sprintf("HitRatio=%d%%", ratio)
		b.Run(name, func(b *testing.B) {
			benchmarkWithHitRatio(b, ratio)
		})
	}
}

// benchmarkWithHitRatio benchmarks cache performance with a specific hit ratio.
// It artificially creates a workload with the specified hit ratio by
// controlling which keys are accessed and which are in the cache.
//
// Parameters:
//   - b: The benchmark context
//   - hitRatio: The target hit ratio as a percentage (0-100)
//
// benchmarkWithHitRatio 使用特定命中率对缓存性能进行基准测试。
// 它通过控制访问哪些键以及哪些键在缓存中，
// 人为地创建具有指定命中率的工作负载。
//
// 参数:
//   - b: 基准测试上下文
//   - hitRatio: 目标命中率（百分比，0-100）
func benchmarkWithHitRatio(b *testing.B, hitRatio int) {
	// Create cache
	// 创建缓存
	cacheInstance, err := cache.NewWithOptions("hitratio-cache",
		cache.WithMaxEntryCount(10000),
		cache.WithShards(16),
		cache.WithTTL(time.Hour),
		cache.WithMetricsEnabled(false),
	)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheInstance.Close()

	ctx := context.Background()

	// Generate test data
	// 生成测试数据
	const keyCount = 10000
	keys := make([]string, keyCount)
	values := make([][]byte, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
		values[i] = make([]byte, 128)
		rand.Read(values[i])
	}

	// Calculate how many keys to preload based on hit ratio
	// 根据命中率计算要预加载的键数量
	keysToPreload := keyCount * hitRatio / 100

	// Preload cache to achieve the target hit ratio
	// 预加载缓存以达到目标命中率
	for i := 0; i < keysToPreload; i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Choose a key that will either hit or miss based on our preloading
			// 选择一个键，根据我们的预加载情况，它将命中或未命中
			key := keys[i%keyCount]

			// Get operation
			// 获取操作
			_, exists, err := cacheInstance.Get(ctx, key)
			if err != nil {
				b.Fatalf("Failed to get from cache: %v", err)
			}

			// If it was a miss, set the key for future hits
			// 如果未命中，设置键以便将来命中
			if !exists {
				err := cacheInstance.Set(ctx, key, values[i%len(values)], time.Hour)
				if err != nil {
					b.Fatalf("Failed to set cache: %v", err)
				}
			}

			i++
		}
	})
}

// BenchmarkValueSize benchmarks cache performance with different value sizes.
// It tests how the cache performs when storing values of different sizes,
// which helps understand memory usage and performance trade-offs.
//
// Parameters:
//   - b: The benchmark context
//
// BenchmarkValueSize 使用不同值大小对缓存性能进行基准测试。
// 它测试缓存在存储不同大小的值时的性能表现，
// 这有助于了解内存使用和性能权衡。
//
// 参数:
//   - b: 基准测试上下文
func BenchmarkValueSize(b *testing.B) {
	valueSizes := []int{64, 256, 1024, 4096, 16384}

	for _, size := range valueSizes {
		name := fmt.Sprintf("ValueSize=%d", size)
		b.Run(name, func(b *testing.B) {
			benchmarkWithValueSize(b, size)
		})
	}
}

// benchmarkWithValueSize benchmarks cache performance with a specific value size.
// It measures the impact of value size on cache operations, which is important
// for understanding how the cache performs with different types of data.
//
// Parameters:
//   - b: The benchmark context
//   - valueSize: The size of values to use in bytes
//
// benchmarkWithValueSize 使用特定值大小对缓存性能进行基准测试。
// 它测量值大小对缓存操作的影响，这对于理解缓存
// 如何处理不同类型的数据非常重要。
//
// 参数:
//   - b: 基准测试上下文
//   - valueSize: 要使用的值的大小（字节）
func benchmarkWithValueSize(b *testing.B, valueSize int) {
	// Create cache
	// 创建缓存
	cacheInstance, err := cache.NewWithOptions("valuesize-cache",
		cache.WithMaxEntryCount(10000),
		cache.WithShards(16),
		cache.WithTTL(time.Hour),
		cache.WithMetricsEnabled(false),
	)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheInstance.Close()

	ctx := context.Background()

	// Generate test data with the specified value size
	// 生成具有指定值大小的测试数据
	const keyCount = 1000
	keys := make([]string, keyCount)
	values := make([][]byte, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
		values[i] = make([]byte, valueSize)
		rand.Read(values[i])
	}

	// Preload some data
	// 预加载一些数据
	for i := 0; i < keyCount/2; i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Alternate between get and set operations
			// 在获取和设置操作之间交替
			if i%2 == 0 {
				// Get operation
				// 获取操作
				key := keys[i%keyCount]
				_, _, err := cacheInstance.Get(ctx, key)
				if err != nil {
					b.Fatalf("Failed to get from cache: %v", err)
				}
			} else {
				// Set operation
				// 设置操作
				key := keys[i%keyCount]
				err := cacheInstance.Set(ctx, key, values[i%keyCount], time.Hour)
				if err != nil {
					b.Fatalf("Failed to set cache: %v", err)
				}
			}
			i++
		}
	})
}
