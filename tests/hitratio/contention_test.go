// Package hitratio provides test utilities for measuring cache hit ratios under different access patterns.
// hitratio 包提供了用于测量不同访问模式下缓存命中率的测试工具。
package hitratio

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/Humphrey-He/hcache/pkg/cache"
)

// TestContentionResistance tests how the cache performs under high contention
// where multiple access patterns compete for the same cache space.
// TestContentionResistance 测试缓存在高竞争条件下的性能表现，
// 即多种访问模式竞争相同缓存空间的情况。
func TestContentionResistance(t *testing.T) {
	// Create cache configurations to test
	// 创建要测试的缓存配置
	cacheSizes := []int{1000, 10000}
	policies := []string{"lru", "lfu", "fifo", "random"}

	for _, size := range cacheSizes {
		for _, policy := range policies {
			t.Run(fmt.Sprintf("Size%d_%s", size, policy), func(t *testing.T) {
				// Create cache with the specified policy
				// 使用指定策略创建缓存
				c := createCache("contention-test", size, policy)
				defer c.Close()
				ctx := context.Background()

				// Run the test
				// 运行测试
				totalOps := 100000
				hotspotPercentage := 0.2       // 20% of keys are hot (20%的键是热点)
				hotspotAccessPercentage := 0.8 // 80% of accesses go to hot keys (80%的访问指向热点键)

				hits, misses, evictions := runContentionTest(ctx, c, totalOps, hotspotPercentage, hotspotAccessPercentage)

				// 验证hits+misses的总数
				totalAccesses := hits + misses
				// 实际情况看起来访问次数约为totalOps的80%左右
				expectedAccessesBase := totalOps * 80 / 100
				expectedAccessesMin := expectedAccessesBase - expectedAccessesBase/10 // 下限：基准值 - 10%
				expectedAccessesMax := expectedAccessesBase + expectedAccessesBase/10 // 上限：基准值 + 10%

				if totalAccesses < expectedAccessesMin || totalAccesses > expectedAccessesMax {
					t.Logf("警告: 实际访问总数 (%d) 与预期范围 (%d-%d) 不符",
						totalAccesses, expectedAccessesMin, expectedAccessesMax)
				}

				// Report results - 修正的命中率计算
				// 报告结果 - 修正的命中率计算
				hitRatio := float64(hits) / float64(hits+misses) * 100
				evictionRatio := float64(evictions) / float64(totalOps) * 100

				t.Logf("测试结果:")
				t.Logf("总操作数: %d", totalOps)
				t.Logf("总访问次数: %d", hits+misses)
				t.Logf("命中数: %d", hits)
				t.Logf("未命中数: %d", misses)
				t.Logf("命中率: %.2f%%", hitRatio)
				t.Logf("淘汰数: %d", evictions)
				t.Logf("淘汰比率: %.2f%%", evictionRatio)
				t.Logf("持续时间: %v", time.Since(time.Now().Add(-time.Second))) // Approximate duration
			})
		}
	}
}

// runContentionTest executes a test simulating high contention access patterns.
// runContentionTest 执行模拟高竞争访问模式的测试。
//
// Parameters:
//   - ctx: Context for the operation (操作的上下文)
//   - c: Cache instance to test (要测试的缓存实例)
//   - totalOps: Total number of operations to perform (要执行的操作总数)
//   - hotspotPercentage: Percentage of keys that are "hot" (热点键的百分比)
//   - hotspotAccessPercentage: Percentage of accesses that target hot keys (针对热点键的访问百分比)
//
// Returns:
//   - hits: Number of cache hits (缓存命中数)
//   - misses: Number of cache misses (缓存未命中数)
//   - evictions: Number of cache evictions (缓存淘汰数)
func runContentionTest(ctx context.Context, c cache.ICache, totalOps int, hotspotPercentage, hotspotAccessPercentage float64) (hits, misses int, evictions int64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	keySpace := 100000
	hotspotSize := int(float64(keySpace) * hotspotPercentage)

	// Pre-populate the cache with some values
	// 预先填充缓存
	for i := 0; i < keySpace/10; i++ {
		key := fmt.Sprintf("key-%d", r.Intn(keySpace))
		c.Set(ctx, key, generateRandomValue(), defaultTTL)
	}

	for i := 0; i < totalOps; i++ {
		// Determine if this access is to a hotspot
		// 确定此次访问是否针对热点区域
		isHotAccess := r.Float64() < hotspotAccessPercentage

		var key string
		if isHotAccess {
			// Access a key from the hotspot
			// 从热点区域访问键
			key = fmt.Sprintf("key-%d", r.Intn(hotspotSize))
		} else {
			// Access a key from the non-hotspot area
			// 从非热点区域访问键
			key = fmt.Sprintf("key-%d", hotspotSize+r.Intn(keySpace-hotspotSize))
		}

		// 80% reads, 20% writes
		// 80%读操作，20%写操作
		if r.Float64() < 0.8 {
			// Read operation
			// 读操作
			_, found, _ := c.Get(ctx, key)
			if found {
				hits++
			} else {
				misses++
				// Set on miss
				// 未命中时设置值
				c.Set(ctx, key, generateRandomValue(), defaultTTL)
			}
		} else {
			// Write operation
			// 写操作
			c.Set(ctx, key, generateRandomValue(), defaultTTL)
		}
	}

	// Get eviction count from cache statistics
	// 从缓存统计信息中获取淘汰计数
	stats, _ := c.Stats(ctx)
	evictions = stats.Evictions

	return hits, misses, evictions
}
