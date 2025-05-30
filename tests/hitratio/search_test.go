// Package hitratio provides test utilities for measuring cache hit ratios under different access patterns.
// hitratio 包提供了用于测量不同访问模式下缓存命中率的测试工具。
package hitratio

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/noobtrump/hcache/pkg/cache"
)

// TestSearchPattern tests cache performance with search-like access patterns
// where data is accessed in a pattern similar to search engine queries.
// TestSearchPattern 测试缓存在类搜索访问模式下的性能表现，
// 即数据访问模式类似于搜索引擎查询的情况。
func TestSearchPattern(t *testing.T) {
	// Create cache configurations to test
	// 创建要测试的缓存配置
	cacheSizes := []int{1000, 10000}
	policies := []string{"lru", "lfu", "fifo", "random"}

	for _, size := range cacheSizes {
		for _, policy := range policies {
			t.Run(fmt.Sprintf("Size%d_%s", size, policy), func(t *testing.T) {
				// Create cache with the specified policy
				// 使用指定策略创建缓存
				c := createCache("search-test", size, policy)
				defer c.Close()
				ctx := context.Background()

				// Run the test
				// 运行测试
				totalOps := 100000
				// Search pattern: few very popular terms, many rare terms
				// 搜索模式：少量非常热门的术语，大量罕见的术语
				popularTerms := 100    // Number of popular search terms (热门搜索词的数量)
				rareProbability := 0.3 // Probability of accessing a rare term (访问罕见词的概率)

				hits, misses, evictions := runSearchTest(ctx, c, totalOps, popularTerms, rareProbability)

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

// runSearchTest executes a test simulating search engine access patterns.
// runSearchTest 执行模拟搜索引擎访问模式的测试。
//
// Parameters:
//   - ctx: Context for the operation (操作的上下文)
//   - c: Cache instance to test (要测试的缓存实例)
//   - totalOps: Total number of operations to perform (要执行的操作总数)
//   - popularTerms: Number of popular search terms (热门搜索词的数量)
//   - rareProbability: Probability of accessing a rare term (访问罕见词的概率)
//
// Returns:
//   - hits: Number of cache hits (缓存命中数)
//   - misses: Number of cache misses (缓存未命中数)
//   - evictions: Number of cache evictions (缓存淘汰数)
func runSearchTest(ctx context.Context, c cache.ICache, totalOps, popularTerms int, rareProbability float64) (hits, misses int, evictions int64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	keySpace := 100000 // Total possible search terms (可能的搜索词总数)

	// Pre-populate the cache with some values
	// 预先填充缓存
	for i := 0; i < keySpace/10; i++ {
		key := fmt.Sprintf("term-%d", r.Intn(keySpace))
		c.Set(ctx, key, generateRandomValue(), defaultTTL)
	}

	for i := 0; i < totalOps; i++ {
		var key string

		// Determine if this is a rare term access
		// 确定这是否是罕见词访问
		isRare := r.Float64() < rareProbability

		if isRare {
			// Access a rare term
			// 访问罕见词
			key = fmt.Sprintf("term-%d", popularTerms+r.Intn(keySpace-popularTerms))
		} else {
			// Access a popular term with zipfian distribution
			// 使用齐普夫分布访问热门词
			rank := zipfRank(r, popularTerms, 1.1)
			key = fmt.Sprintf("term-%d", rank)
		}

		// Search queries are mostly reads
		// 搜索查询主要是读操作
		_, found, _ := c.Get(ctx, key)
		if found {
			hits++
		} else {
			misses++
			// Set on miss (simulating caching the search result)
			// 未命中时设置值（模拟缓存搜索结果）
			c.Set(ctx, key, generateRandomValue(), defaultTTL)
		}
	}

	stats, _ := c.Stats(ctx)
	evictions = stats.Evictions
	return hits, misses, evictions
}
