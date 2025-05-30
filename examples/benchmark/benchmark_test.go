// Package benchmark provides comprehensive benchmarks for the HCache library.
// It tests various cache configurations, access patterns, and eviction policies
// to measure performance characteristics under different workloads.
//
// Package benchmark 为HCache库提供全面的基准测试。
// 它测试各种缓存配置、访问模式和淘汰策略，以测量不同工作负载下的性能特征。
package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/noobtrump/hcache/pkg/cache"
)

// BenchmarkCache runs a series of benchmarks on the cache with different configurations.
// It tests various combinations of cache sizes and shard counts to measure how these
// parameters affect performance.
//
// BenchmarkCache 使用不同配置对缓存运行一系列基准测试。
// 它测试缓存大小和分片数量的各种组合，以测量这些参数如何影响性能。
func BenchmarkCache(b *testing.B) {
	// Run benchmarks with different cache configurations
	// 使用不同的缓存配置运行基准测试
	cacheSizes := []int{1000, 10000, 100000}
	shardCounts := []int{1, 4, 16, 64}

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

	// Generate test data - keys and random byte values
	// 生成测试数据 - 键和随机字节值
	keys := make([]string, cacheSize)
	values := make([][]byte, cacheSize)
	for i := 0; i < cacheSize; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
		values[i] = make([]byte, 1024)
		rand.Read(values[i])
	}

	// Run individual benchmarks for different operations
	// 为不同操作运行单独的基准测试
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

	// Run mixed workload benchmarks with different read/write ratios
	// 运行具有不同读/写比率的混合工作负载基准测试
	b.Run("Mixed/Read80Write20", func(b *testing.B) {
		benchmarkMixed(b, ctx, cacheInstance, keys, values, 80)
	})

	b.Run("Mixed/Read50Write50", func(b *testing.B) {
		benchmarkMixed(b, ctx, cacheInstance, keys, values, 50)
	})

	b.Run("Mixed/Read20Write80", func(b *testing.B) {
		benchmarkMixed(b, ctx, cacheInstance, keys, values, 20)
	})

	// Run realistic workload simulation with zipfian distribution
	// 使用齐普夫分布运行真实工作负载模拟
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
	// Preload cache with a subset of keys to ensure hits
	// 预加载缓存的键子集以确保命中
	for i := 0; i < len(keys) && i < 1000; i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
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
	b.ResetTimer()
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
	for i := 0; i < len(keys) && i < 1000; i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
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
	for i := 0; i < len(keys) && i < 1000; i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	b.ResetTimer()
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
				_, _, err := cacheInstance.Get(ctx, key)
				if err != nil {
					b.Fatalf("Failed to get from cache: %v", err)
				}
			} else {
				err := cacheInstance.Set(ctx, key, values[i%len(values)], time.Hour)
				if err != nil {
					b.Fatalf("Failed to set cache: %v", err)
				}
			}
			i++
		}
	})
}

// benchmarkZipfianAccess benchmarks cache access with a zipfian distribution.
// This simulates real-world access patterns where some keys are much more popular than others,
// which is common in many applications (e.g., social media, news sites).
//
// Parameters:
//   - b: The benchmark context
//   - ctx: The context for cache operations
//   - cacheInstance: The cache instance to benchmark
//   - keys: The keys to use for the benchmark
//   - values: The values to use for the benchmark
//
// benchmarkZipfianAccess 使用齐普夫分布对缓存访问进行基准测试。
// 这模拟了真实世界的访问模式，其中某些键比其他键更受欢迎，
// 这在许多应用程序中很常见（例如，社交媒体、新闻网站）。
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
	for i := 0; i < len(keys) && i < 1000; i++ {
		err := cacheInstance.Set(ctx, keys[i], values[i], time.Hour)
		if err != nil {
			b.Fatalf("Failed to set cache: %v", err)
		}
	}

	// Create zipfian distribution
	// 创建齐普夫分布
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), 1.5, 1.0, uint64(len(keys)-1))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Get a zipfian distributed index
			// 获取齐普夫分布的索引
			idx := zipf.Uint64()
			key := keys[idx]

			// 80% reads, 20% writes
			// 80%读取，20%写入
			if rand.Intn(100) < 80 {
				_, _, err := cacheInstance.Get(ctx, key)
				if err != nil {
					b.Fatalf("Failed to get from cache: %v", err)
				}
			} else {
				err := cacheInstance.Set(ctx, key, values[idx%uint64(len(values))], time.Hour)
				if err != nil {
					b.Fatalf("Failed to set cache: %v", err)
				}
			}
		}
	})
}

// BenchmarkEvictionPolicies benchmarks different eviction policies.
// It compares the performance of LRU, LFU, FIFO, and random eviction
// strategies under the same workload.
//
// BenchmarkEvictionPolicies 对不同的淘汰策略进行基准测试。
// 它比较了LRU、LFU、FIFO和随机淘汰策略在相同工作负载下的性能。
func BenchmarkEvictionPolicies(b *testing.B) {
	evictionPolicies := []string{"lru", "lfu", "fifo", "random"}

	for _, policy := range evictionPolicies {
		b.Run(policy, func(b *testing.B) {
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
	cacheInstance, err := cache.NewWithOptions("benchmark-cache",
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
	keys := make([]string, 10000) // 10x the cache size to force evictions
	values := make([][]byte, 10000)
	for i := 0; i < 10000; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
		values[i] = make([]byte, 1024)
		rand.Read(values[i])
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Alternate between setting and getting
			// 在设置和获取之间交替
			if i%2 == 0 {
				// Set operation
				// 设置操作
				key := keys[i%len(keys)]
				err := cacheInstance.Set(ctx, key, values[i%len(values)], time.Hour)
				if err != nil {
					b.Fatalf("Failed to set cache: %v", err)
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

// BenchmarkTTL benchmarks TTL expiration performance.
// It tests how the cache performs when items are constantly
// expiring due to short TTLs.
//
// BenchmarkTTL 对TTL过期性能进行基准测试。
// 它测试当项目由于短TTL而不断过期时缓存的性能表现。
func BenchmarkTTL(b *testing.B) {
	// Create cache with very short TTL
	// 创建具有非常短的TTL的缓存
	cacheInstance, err := cache.NewWithOptions("benchmark-cache",
		cache.WithMaxEntryCount(10000),
		cache.WithShards(16),
		cache.WithTTL(time.Millisecond*100), // Very short TTL
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
		values[i] = make([]byte, 1024)
		rand.Read(values[i])
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Every 100 iterations, set a batch of keys
			// 每100次迭代，设置一批键
			if i%100 == 0 {
				for j := 0; j < 10; j++ {
					key := keys[(i+j)%len(keys)]
					err := cacheInstance.Set(ctx, key, values[(i+j)%len(values)], time.Millisecond*100)
					if err != nil {
						b.Fatalf("Failed to set cache: %v", err)
					}
				}
				// Sleep to allow some keys to expire
				// 休眠以允许一些键过期
				time.Sleep(time.Millisecond * 50)
			}

			// Get operation
			// 获取操作
			key := keys[i%len(keys)]
			_, _, err := cacheInstance.Get(ctx, key)
			if err != nil {
				b.Fatalf("Failed to get from cache: %v", err)
			}
			i++
		}
	})
}

// TestDummy is a simple test to verify that the benchmark package works.
// This is just a placeholder test to ensure the package can be built and tested.
//
// TestDummy 是一个简单的测试，用于验证基准测试包是否正常工作。
// 这只是一个占位符测试，以确保包可以构建和测试。
func TestDummy(t *testing.T) {
	// This is just a placeholder test
	// 这只是一个占位符测试
}
