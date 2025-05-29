// Package hitratio provides test utilities for measuring cache hit ratios under different access patterns.
// hitratio 包提供了用于测量不同访问模式下缓存命中率的测试工具。
package hitratio

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/yourusername/hcache/pkg/cache"
)

// TestLoopingPattern tests cache performance with looping access patterns
// where the same set of data is accessed repeatedly in loops.
// TestLoopingPattern 测试缓存在循环访问模式下的性能表现，
// 即相同的数据集在循环中被重复访问的情况。
func TestLoopingPattern(t *testing.T) {
	// Create cache configurations to test
	// 创建要测试的缓存配置
	cacheSizes := []int{1000, 10000}
	policies := []string{"lru", "lfu", "fifo", "random"}

	for _, size := range cacheSizes {
		for _, policy := range policies {
			t.Run(fmt.Sprintf("Size%d_%s", size, policy), func(t *testing.T) {
				// Create cache with the specified policy
				// 使用指定策略创建缓存
				c := createCache("looping-test", size, policy)
				defer c.Close()
				ctx := context.Background()

				// Run the test
				// 运行测试
				totalOps := 100000
				loopSize := 5000     // Size of the loop (循环大小)
				loopVariation := 0.1 // 10% variation in the loop pattern (循环模式中的10%变化)

				hits, misses, evictions := runLoopingTest(ctx, c, totalOps, loopSize, loopVariation)

				// Report results
				// 报告结果
				hitRatio := float64(hits) / float64(totalOps) * 100
				evictionRatio := float64(evictions) / float64(totalOps) * 100

				t.Logf("测试结果:")
				t.Logf("总操作数: %d", totalOps)
				t.Logf("命中数: %d", hits)
				t.Logf("未命中数: %d", misses)
				t.Logf("命中率: %.2f%%", hitRatio)
				t.Logf("淘汰数: %d", evictions)
				t.Logf("淘汰比率: %.2f%%", evictionRatio)
				t.Logf("持续时间: %v", time.Since(time.Now().Add(-time.Second))) // Approximate duration (近似持续时间)
			})
		}
	}
}

// runLoopingTest executes a test simulating looping access patterns.
// runLoopingTest 执行模拟循环访问模式的测试。
//
// Parameters:
//   - ctx: Context for the operation (操作的上下文)
//   - c: Cache instance to test (要测试的缓存实例)
//   - totalOps: Total number of operations to perform (要执行的操作总数)
//   - loopSize: Size of the access loop (访问循环的大小)
//   - loopVariation: Probability of deviating from the loop pattern (偏离循环模式的概率)
//
// Returns:
//   - hits: Number of cache hits (缓存命中数)
//   - misses: Number of cache misses (缓存未命中数)
//   - evictions: Number of cache evictions (缓存淘汰数)
func runLoopingTest(ctx context.Context, c cache.ICache, totalOps, loopSize int, loopVariation float64) (hits, misses int, evictions int64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	keySpace := 100000

	// Pre-populate the cache with some values
	// 预先填充缓存
	for i := 0; i < keySpace/10; i++ {
		key := fmt.Sprintf("key-%d", r.Intn(keySpace))
		c.Set(ctx, key, generateRandomValue(), defaultTTL)
	}

	// Create a loop pattern
	// 创建循环模式
	loopKeys := make([]string, loopSize)
	for i := 0; i < loopSize; i++ {
		loopKeys[i] = fmt.Sprintf("key-%d", r.Intn(keySpace))
	}

	loopPos := 0
	for i := 0; i < totalOps; i++ {
		// Determine if we should vary from the loop
		// 确定是否应该偏离循环
		shouldVary := r.Float64() < loopVariation

		var key string
		if shouldVary {
			// Access a random key outside the loop
			// 访问循环外的随机键
			key = fmt.Sprintf("key-%d", r.Intn(keySpace))
		} else {
			// Access the next key in the loop
			// 访问循环中的下一个键
			key = loopKeys[loopPos]
			loopPos = (loopPos + 1) % loopSize
		}

		// Mostly reads in loops
		// 循环中主要是读操作
		_, found, _ := c.Get(ctx, key)
		if found {
			hits++
		} else {
			misses++
			// Set on miss
			// 未命中时设置值
			c.Set(ctx, key, generateRandomValue(), defaultTTL)
		}
	}

	stats, _ := c.Stats(ctx)
	evictions = stats.Evictions
	return hits, misses, evictions
}
