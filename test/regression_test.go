// Package test provides regression and benchmark tests for the HCache library.
// It contains tests that verify the correctness and performance of the cache
// implementation under various conditions and workloads.
//
// Package test 为HCache库提供回归测试和基准测试。
// 它包含在各种条件和工作负载下验证缓存实现的正确性和性能的测试。
package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Humphrey-He/hcache/pkg/cache"
)

// createTestCache creates a mock cache instance for testing purposes.
// It provides a consistent way to create cache instances across test cases.
//
// Parameters:
//   - t: The testing context
//   - name: A name for the cache instance
//   - maxEntries: Maximum number of entries the cache can hold
//
// Returns:
//   - cache.ICache: A new cache instance ready for testing
//
// createTestCache 创建用于测试目的的模拟缓存实例。
// 它提供了一种在测试用例中创建缓存实例的一致方法。
//
// 参数:
//   - t: 测试上下文
//   - name: 缓存实例的名称
//   - maxEntries: 缓存可以容纳的最大条目数
//
// 返回:
//   - cache.ICache: 准备好进行测试的新缓存实例
func createTestCache(t *testing.T, name string, maxEntries int) cache.ICache {
	// Create a mock cache directly
	// 直接创建一个模拟缓存
	return cache.NewMockCache(name, maxEntries, "lru")
}

// TestRegressionCacheConcurrency tests that the cache behaves correctly under concurrent access.
// This test verifies that the cache maintains data consistency when accessed by multiple
// goroutines simultaneously, simulating a high-concurrency production environment.
//
// Parameters:
//   - t: The testing context
//
// TestRegressionCacheConcurrency 测试缓存在并发访问下的行为是否正确。
// 此测试验证当多个goroutine同时访问缓存时，缓存能否保持数据一致性，模拟高并发生产环境。
//
// 参数:
//   - t: 测试上下文
func TestRegressionCacheConcurrency(t *testing.T) {
	// Create cache
	// 创建缓存
	cacheInstance := createTestCache(t, "regression-cache", 10000)
	defer cacheInstance.Close()

	ctx := context.Background()

	// Test concurrent access
	// 测试并发访问
	const numGoroutines = 100
	const numOperations = 1000

	// Create a channel to signal completion
	// 创建一个通道以发出完成信号
	done := make(chan struct{}, numGoroutines)

	// Start goroutines
	// 启动goroutine
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()

			// Perform operations
			// 执行操作
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key:%d:%d", id, j)
				value := fmt.Sprintf("value:%d:%d", id, j)

				// Set value
				// 设置值
				err := cacheInstance.Set(ctx, key, value, time.Hour)
				if err != nil {
					t.Errorf("Failed to set cache: %v", err)
					return
				}

				// Get value
				// 获取值
				val, exists, err := cacheInstance.Get(ctx, key)
				if err != nil {
					t.Errorf("Failed to get from cache: %v", err)
					return
				}
				if !exists {
					t.Errorf("Expected key %s to exist", key)
					return
				}
				if val != value {
					t.Errorf("Expected value %s, got %s", value, val)
					return
				}

				// Delete value (for some keys)
				// 删除值（对于某些键）
				if j%10 == 0 {
					_, err := cacheInstance.Delete(ctx, key)
					if err != nil {
						t.Errorf("Failed to delete from cache: %v", err)
						return
					}

					// Verify deletion
					// 验证删除
					_, exists, err = cacheInstance.Get(ctx, key)
					if err != nil {
						t.Errorf("Failed to get from cache after deletion: %v", err)
						return
					}
					if exists {
						t.Errorf("Expected key %s to be deleted", key)
						return
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check cache stats
	// 检查缓存统计信息
	stats, err := cacheInstance.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get cache stats: %v", err)
	}

	// Verify that the stats make sense
	// 验证统计信息是否合理
	if stats.Hits == 0 {
		t.Errorf("Expected cache hits to be non-zero")
	}
	if stats.Misses == 0 {
		t.Errorf("Expected cache misses to be non-zero")
	}
	if stats.EntryCount == 0 {
		t.Errorf("Expected cache entry count to be non-zero")
	}
}

// TestRegressionCacheExpiration tests that the cache correctly expires entries.
// This test verifies that entries are automatically removed from the cache
// after their TTL (Time To Live) has elapsed.
//
// Parameters:
//   - t: The testing context
//
// TestRegressionCacheExpiration 测试缓存是否正确地使条目过期。
// 此测试验证条目在其TTL（存活时间）过后是否会自动从缓存中移除。
//
// 参数:
//   - t: 测试上下文
func TestRegressionCacheExpiration(t *testing.T) {
	// Create cache with short TTL
	// 创建具有短TTL的缓存
	cacheInstance := createTestCache(t, "regression-cache", 1000)
	defer cacheInstance.Close()

	ctx := context.Background()

	// Add some entries
	// 添加一些条目
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("value:%d", i)
		err := cacheInstance.Set(ctx, key, value, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to set cache: %v", err)
		}
	}

	// Verify that entries exist
	// 验证条目存在
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key:%d", i)
		_, exists, err := cacheInstance.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get from cache: %v", err)
		}
		if !exists {
			t.Errorf("Expected key %s to exist", key)
		}
	}

	// Wait for entries to expire
	// 等待条目过期
	time.Sleep(200 * time.Millisecond)

	// Verify that entries have expired
	// 验证条目已过期
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key:%d", i)
		_, exists, err := cacheInstance.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get from cache: %v", err)
		}
		if exists {
			t.Errorf("Expected key %s to have expired", key)
		}
	}
}

// TestRegressionCacheEviction tests that the cache correctly evicts entries when full.
// This test verifies that the cache enforces its maximum size limit by removing
// entries according to its eviction policy when new entries are added to a full cache.
//
// Parameters:
//   - t: The testing context
//
// TestRegressionCacheEviction 测试缓存在满时是否正确地淘汰条目。
// 此测试验证缓存是否通过在向满缓存添加新条目时根据其淘汰策略移除条目来强制执行其最大大小限制。
//
// 参数:
//   - t: 测试上下文
func TestRegressionCacheEviction(t *testing.T) {
	// Create a small cache
	// 创建一个小缓存
	cacheInstance := createTestCache(t, "regression-cache", 100)
	defer cacheInstance.Close()

	ctx := context.Background()

	// Add more entries than the cache can hold
	// 添加超过缓存可容纳的条目
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("key:%d", i)
		value := fmt.Sprintf("value:%d", i)
		err := cacheInstance.Set(ctx, key, value, time.Hour)
		if err != nil {
			// Ignore errors from eviction policy
			// 忽略来自淘汰策略的错误
			if err.Error() != "cache: cache is full" {
				t.Fatalf("Failed to set cache: %v", err)
			}
		}
	}

	// Get cache stats
	// 获取缓存统计信息
	stats, err := cacheInstance.Stats(ctx)
	if err != nil {
		t.Fatalf("Failed to get cache stats: %v", err)
	}

	// Verify that the cache size is at or below the limit
	// 验证缓存大小是否在限制范围内或以下
	if stats.EntryCount > 100 {
		t.Errorf("Expected cache entry count to be at most 100, got %d", stats.EntryCount)
	}
}
