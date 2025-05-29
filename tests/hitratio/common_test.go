// Package hitratio provides test utilities for measuring cache hit ratios under different access patterns.
// hitratio 包提供了用于测量不同访问模式下缓存命中率的测试工具。
package hitratio

import (
	"math/rand"
	"time"

	"github.com/yourusername/hcache/pkg/cache"
)

// Constants for tests
// 测试常量
const (
	defaultTTL = 10 * time.Minute // Default time-to-live for cache entries (缓存条目的默认生存时间)
	valueSize  = 1024             // Size of generated random values in bytes (生成的随机值大小，以字节为单位)
)

// generateRandomValue generates a random byte array for cache values.
// generateRandomValue 生成用于缓存值的随机字节数组。
//
// Returns:
//   - []byte: A random byte array of size valueSize
//     返回一个大小为 valueSize 的随机字节数组
func generateRandomValue() []byte {
	value := make([]byte, valueSize)
	rand.Read(value)
	return value
}

// zipfRank returns a rank according to Zipf's law.
// zipfRank 根据齐普夫定律返回一个排名。
//
// Parameters:
//   - r: Random number generator (随机数生成器)
//   - n: Maximum rank value (最大排名值)
//   - s: Zipf distribution parameter (齐普夫分布参数)
//
// Returns:
//   - int: A rank value following Zipf distribution (遵循齐普夫分布的排名值)
func zipfRank(r *rand.Rand, n int, s float64) int {
	// Simple approximation of Zipf distribution
	// 齐普夫分布的简单近似
	x := r.Float64()
	rank := int(float64(n) * (1.0 - x) * (1.0 - x))
	if rank >= n {
		rank = n - 1
	}
	return rank
}

// createCache creates a new cache with the specified policy.
// createCache 创建一个具有指定策略的新缓存。
//
// Parameters:
//   - name: Name of the cache (缓存名称)
//   - size: Maximum size of the cache (缓存最大大小)
//   - policy: Eviction policy to use (使用的淘汰策略)
//
// Returns:
//   - cache.ICache: A new cache instance (新的缓存实例)
func createCache(name string, size int, policy string) cache.ICache {
	return cache.NewMockCache(name, size, policy)
}
