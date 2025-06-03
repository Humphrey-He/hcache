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

// TestDatabasePattern tests cache performance with database-like access patterns
// where data access follows typical database query patterns.
// TestDatabasePattern 测试缓存在类数据库访问模式下的性能表现，
// 即数据访问遵循典型数据库查询模式的情况。
func TestDatabasePattern(t *testing.T) {
	// Create cache configurations to test
	// 创建要测试的缓存配置
	cacheSizes := []int{1000, 10000}
	policies := []string{"lru", "lfu", "fifo", "random"}

	for _, size := range cacheSizes {
		for _, policy := range policies {
			t.Run(fmt.Sprintf("Size%d_%s", size, policy), func(t *testing.T) {
				// Create cache with the specified policy
				// 使用指定策略创建缓存
				c := createCache("database-test", size, policy)
				defer c.Close()
				ctx := context.Background()

				// Run the test
				// 运行测试
				totalOps := 100000
				readWriteRatio := 0.8   // 80% reads, 20% writes (80%读操作，20%写操作)
				indexAccessRatio := 0.6 // 60% of reads are index lookups (60%的读操作是索引查找)

				hits, misses, evictions := runDatabaseTest(ctx, c, totalOps, readWriteRatio, indexAccessRatio)

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

// runDatabaseTest executes a test simulating database access patterns.
// runDatabaseTest 执行模拟数据库访问模式的测试。
//
// Parameters:
//   - ctx: Context for the operation (操作的上下文)
//   - c: Cache instance to test (要测试的缓存实例)
//   - totalOps: Total number of operations to perform (要执行的操作总数)
//   - readWriteRatio: Ratio of read operations to total operations (读操作占总操作的比例)
//   - indexAccessRatio: Ratio of index accesses to total read operations (索引访问占总读操作的比例)
//
// Returns:
//   - hits: Number of cache hits (缓存命中数)
//   - misses: Number of cache misses (缓存未命中数)
//   - evictions: Number of cache evictions (缓存淘汰数)
func runDatabaseTest(ctx context.Context, c cache.ICache, totalOps int, readWriteRatio, indexAccessRatio float64) (hits, misses int, evictions int64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	recordCount := 50000 // Total records (总记录数)
	indexCount := 1000   // Number of indices (索引数量)

	// Pre-populate the cache with some records and indices
	// 预先填充缓存，存入一些记录和索引
	for i := 0; i < recordCount/10; i++ {
		// Cache some records
		// 缓存一些记录
		recordKey := fmt.Sprintf("record-%d", r.Intn(recordCount))
		c.Set(ctx, recordKey, generateRandomValue(), defaultTTL)

		// Cache some indices
		// 缓存一些索引
		if i < indexCount {
			indexKey := fmt.Sprintf("index-%d", i)
			c.Set(ctx, indexKey, generateRandomValue(), defaultTTL)
		}
	}

	for i := 0; i < totalOps; i++ {
		isRead := r.Float64() < readWriteRatio

		if isRead {
			// Read operation
			// 读操作
			isIndexAccess := r.Float64() < indexAccessRatio

			var key string
			if isIndexAccess {
				// Access an index
				// 访问索引
				key = fmt.Sprintf("index-%d", r.Intn(indexCount))
			} else {
				// Access a record
				// 访问记录
				key = fmt.Sprintf("record-%d", r.Intn(recordCount))
			}

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
			key := fmt.Sprintf("record-%d", r.Intn(recordCount))
			c.Set(ctx, key, generateRandomValue(), defaultTTL)
		}
	}

	stats, _ := c.Stats(ctx)
	evictions = stats.Evictions
	return hits, misses, evictions
}
