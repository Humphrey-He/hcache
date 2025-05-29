// Package hitratio provides test utilities for measuring cache hit ratios under different access patterns.
// hitratio 包提供了用于测量不同访问模式下缓存命中率的测试工具。
package hitratio

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// Constants for zipf distribution tests
// Zipf分布测试的常量
const (
	smallCacheSize  = 100
	mediumCacheSize = 1000
	largeCacheSize  = 10000
	keySpaceSize    = 100000
	operationCount  = 100000
)

// Distribution represents the type of key access distribution.
// Distribution 表示键访问分布的类型。
type Distribution string

// Distribution types
// 分布类型
const (
	Uniform  Distribution = "uniform"   // Uniform distribution (均匀分布)
	ZipfLow  Distribution = "zipf-low"  // Low skew Zipf distribution (低偏斜齐普夫分布)
	ZipfHigh Distribution = "zipf-high" // High skew Zipf distribution (高偏斜齐普夫分布)
)

// TestScenario defines a test scenario for hit ratio testing.
// TestScenario 定义了命中率测试的测试场景。
type TestScenario struct {
	Name         string       // Name of the test scenario (测试场景名称)
	CacheSize    int          // Size of the cache (缓存大小)
	Distribution Distribution // Key access distribution (键访问分布)
	Policy       string       // Eviction policy (淘汰策略)
}

// TestResult stores the results of a hit ratio test.
// TestResult 存储命中率测试的结果。
type TestResult struct {
	TotalOperations int           // Total number of operations (操作总数)
	Hits            int           // Number of cache hits (缓存命中数)
	Misses          int           // Number of cache misses (缓存未命中数)
	HitRatio        float64       // Cache hit ratio (缓存命中率)
	Evictions       int64         // Number of cache evictions (缓存淘汰数)
	EvictionRatio   float64       // Cache eviction ratio (缓存淘汰率)
	Duration        time.Duration // Test duration (测试持续时间)
}

// Initialize random number generator
// 初始化随机数生成器
func init() {
	rand.Seed(time.Now().UnixNano())
}

// runHitRatioTest runs a hit ratio test with the given scenario.
// runHitRatioTest 使用给定的场景运行命中率测试。
//
// Parameters:
//   - t: Testing context (测试上下文)
//   - scenario: Test scenario configuration (测试场景配置)
//
// Returns:
//   - TestResult: Results of the test (测试结果)
func runHitRatioTest(t *testing.T, scenario TestScenario) TestResult {
	t.Logf("运行测试: %s", scenario.Name)

	// 创建缓存
	cacheInstance := createCache(scenario.Name, scenario.CacheSize, scenario.Policy)
	defer cacheInstance.Close()

	ctx := context.Background()
	startTime := time.Now()

	// 创建访问分布生成器
	var nextKey func() string
	switch scenario.Distribution {
	case Uniform:
		nextKey = func() string {
			return fmt.Sprintf("key:%d", rand.Intn(keySpaceSize))
		}
	case ZipfLow:
		zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), 1.07, 1.0, uint64(keySpaceSize-1))
		nextKey = func() string {
			return fmt.Sprintf("key:%d", zipf.Uint64())
		}
	case ZipfHigh:
		zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), 1.2, 1.0, uint64(keySpaceSize-1))
		nextKey = func() string {
			return fmt.Sprintf("key:%d", zipf.Uint64())
		}
	default:
		t.Fatalf("未知分布类型: %s", scenario.Distribution)
	}

	// 执行测试操作
	hits := 0
	misses := 0

	for i := 0; i < operationCount; i++ {
		key := nextKey()

		// 90% 的操作是读取，10% 的操作是写入
		if rand.Float64() < 0.9 {
			// 读操作
			_, exists, err := cacheInstance.Get(ctx, key)
			if err != nil {
				t.Fatalf("Get 操作失败: %v", err)
			}
			if exists {
				hits++
			} else {
				misses++
			}
		} else {
			// 写操作
			value := generateRandomValue()
			err := cacheInstance.Set(ctx, key, value, defaultTTL)
			if err != nil {
				t.Fatalf("Set 操作失败: %v", err)
			}
		}
	}

	// 获取统计信息
	stats, err := cacheInstance.Stats(ctx)
	if err != nil {
		t.Fatalf("获取统计信息失败: %v", err)
	}

	duration := time.Since(startTime)

	// 计算结果
	totalOps := hits + misses
	hitRatio := float64(hits) / float64(totalOps)
	evictionRatio := float64(stats.Evictions) / float64(scenario.CacheSize)

	result := TestResult{
		TotalOperations: totalOps,
		Hits:            hits,
		Misses:          misses,
		HitRatio:        hitRatio,
		Evictions:       stats.Evictions,
		EvictionRatio:   evictionRatio,
		Duration:        duration,
	}

	// 输出结果
	t.Logf("测试结果:")
	t.Logf("  总操作数: %d", result.TotalOperations)
	t.Logf("  命中数: %d", result.Hits)
	t.Logf("  未命中数: %d", result.Misses)
	t.Logf("  命中率: %.2f%%", result.HitRatio*100)
	t.Logf("  淘汰数: %d", result.Evictions)
	t.Logf("  淘汰比率: %.2f%%", result.EvictionRatio*100)
	t.Logf("  持续时间: %v", result.Duration)

	return result
}

// TestUniformDistribution tests cache hit ratio with uniform key distribution.
// TestUniformDistribution 测试均匀分布下的缓存命中率。
func TestUniformDistribution(t *testing.T) {
	scenario := TestScenario{
		Name:         "Uniform-LRU",
		CacheSize:    mediumCacheSize,
		Distribution: Uniform,
		Policy:       "lru",
	}
	runHitRatioTest(t, scenario)
}

// TestZipfDistribution tests cache hit ratio with Zipf key distribution.
// TestZipfDistribution 测试Zipf分布下的缓存命中率。
func TestZipfDistribution(t *testing.T) {
	scenarios := []TestScenario{
		{
			Name:         "ZipfLow-LRU",
			CacheSize:    mediumCacheSize,
			Distribution: ZipfLow,
			Policy:       "lru",
		},
		{
			Name:         "ZipfHigh-LRU",
			CacheSize:    mediumCacheSize,
			Distribution: ZipfHigh,
			Policy:       "lru",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			runHitRatioTest(t, scenario)
		})
	}
}

// TestEvictionPolicies tests cache hit ratio with different eviction policies.
// TestEvictionPolicies 测试不同淘汰策略下的缓存命中率。
func TestEvictionPolicies(t *testing.T) {
	policies := []string{"lru", "lfu", "fifo", "random"}

	for _, policy := range policies {
		t.Run(fmt.Sprintf("ZipfHigh-%s", policy), func(t *testing.T) {
			scenario := TestScenario{
				Name:         fmt.Sprintf("ZipfHigh-%s", policy),
				CacheSize:    mediumCacheSize,
				Distribution: ZipfHigh,
				Policy:       policy,
			}
			runHitRatioTest(t, scenario)
		})
	}
}

// TestCacheSizes tests cache hit ratio with different cache sizes.
// TestCacheSizes 测试不同缓存大小下的缓存命中率。
func TestCacheSizes(t *testing.T) {
	sizes := []int{smallCacheSize, mediumCacheSize, largeCacheSize}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size-%d", size), func(t *testing.T) {
			scenario := TestScenario{
				Name:         fmt.Sprintf("ZipfHigh-LRU-Size%d", size),
				CacheSize:    size,
				Distribution: ZipfHigh,
				Policy:       "lru",
			}
			runHitRatioTest(t, scenario)
		})
	}
}

// TestHotSpotOverflow tests cache hit ratio when hot spot data exceeds cache capacity.
// TestHotSpotOverflow 测试热点数据超过缓存容量时的缓存命中率。
func TestHotSpotOverflow(t *testing.T) {
	// 创建一个小缓存，使热点数据超过缓存容量
	scenario := TestScenario{
		Name:         "HotSpotOverflow",
		CacheSize:    100, // 非常小的缓存
		Distribution: ZipfHigh,
		Policy:       "lru",
	}
	runHitRatioTest(t, scenario)
}

// TestConcurrentAccess tests cache hit ratio with concurrent access.
// TestConcurrentAccess 测试并发访问下的缓存命中率。
func TestConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过并发测试")
	}

	scenario := TestScenario{
		Name:         "ConcurrentAccess",
		CacheSize:    mediumCacheSize,
		Distribution: ZipfHigh,
		Policy:       "lru",
	}

	// 创建缓存
	cacheInstance := createCache(scenario.Name, scenario.CacheSize, scenario.Policy)
	defer cacheInstance.Close()

	ctx := context.Background()
	startTime := time.Now()

	// 并发访问
	concurrency := 10
	opsPerGoroutine := operationCount / concurrency
	hitsChan := make(chan int, concurrency)
	missesChan := make(chan int, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			localHits := 0
			localMisses := 0

			// 创建本地随机数生成器
			localRand := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
			zipf := rand.NewZipf(localRand, 1.2, 1.0, uint64(keySpaceSize-1))

			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key:%d", zipf.Uint64())

				// 90% 的操作是读取
				if localRand.Float64() < 0.9 {
					// 读操作
					_, exists, _ := cacheInstance.Get(ctx, key)
					if exists {
						localHits++
					} else {
						localMisses++
						// 未命中时写入
						cacheInstance.Set(ctx, key, generateRandomValue(), defaultTTL)
					}
				} else {
					// 写操作
					cacheInstance.Set(ctx, key, generateRandomValue(), defaultTTL)
				}
			}

			hitsChan <- localHits
			missesChan <- localMisses
		}(i)
	}

	wg.Wait()
	close(hitsChan)
	close(missesChan)

	// 汇总结果
	totalHits := 0
	totalMisses := 0
	for hits := range hitsChan {
		totalHits += hits
	}
	for misses := range missesChan {
		totalMisses += misses
	}

	// 获取统计信息
	stats, _ := cacheInstance.Stats(ctx)
	duration := time.Since(startTime)

	// 计算结果
	totalOps := totalHits + totalMisses
	hitRatio := float64(totalHits) / float64(totalOps) * 100
	evictionRatio := float64(stats.Evictions) / float64(scenario.CacheSize) * 100

	// 输出结果
	t.Logf("并发测试结果 (goroutines=%d):", concurrency)
	t.Logf("  总操作数: %d", totalOps)
	t.Logf("  命中数: %d", totalHits)
	t.Logf("  未命中数: %d", totalMisses)
	t.Logf("  命中率: %.2f%%", hitRatio)
	t.Logf("  淘汰数: %d", stats.Evictions)
	t.Logf("  淘汰比率: %.2f%%", evictionRatio)
	t.Logf("  持续时间: %v", duration)
}
