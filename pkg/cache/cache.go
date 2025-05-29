// Package cache provides a high-performance, thread-safe local caching implementation.
// It supports multiple eviction policies, TTL-based expiration, and detailed performance metrics.
// The package is designed to be used as a building block for applications that need fast data access.
//
// Package cache 提供高性能、线程安全的本地缓存实现。
// 它支持多种淘汰策略、基于TTL的过期机制，并提供详细的性能指标。
// 该包旨在作为需要快速数据访问的应用程序的基础构建块。
package cache

import (
	"context"
	"time"
)

// ICache defines the interface for the cache.
// All methods are thread-safe and can be called concurrently.
//
// ICache 定义缓存的接口。
// 所有方法都是线程安全的，可以并发调用。
type ICache interface {
	// Get retrieves a value from the cache.
	// Returns the value and a boolean indicating whether the key was found.
	// If the key is not found or has expired, (nil, false, nil) is returned.
	// An error is returned if the retrieval fails for any reason.
	//
	// Get 从缓存中检索值。
	// 返回值和一个布尔值，指示是否找到了键。
	// 如果未找到键或键已过期，则返回 (nil, false, nil)。
	// 如果检索由于任何原因失败，则返回错误。
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - key: The key to retrieve
	//
	// Returns:
	//   - interface{}: The cached value if found
	//   - bool: True if the key was found and is valid
	//   - error: Error if the retrieval operation failed
	Get(ctx context.Context, key string) (interface{}, bool, error)

	// Set adds a value to the cache with the specified TTL.
	// If the key already exists, its value is updated.
	// If ttl is 0, the default TTL from the configuration is used.
	// If ttl is negative, the entry does not expire.
	//
	// Set 将值添加到缓存中，并指定TTL。
	// 如果键已存在，则更新其值。
	// 如果ttl为0，则使用配置中的默认TTL。
	// 如果ttl为负数，则条目不会过期。
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - key: The key under which to store the value
	//   - value: The value to store
	//   - ttl: Time-to-live for the entry
	//
	// Returns:
	//   - error: Error if the set operation failed
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache.
	// Returns true if the key was found and removed, false if the key was not found.
	//
	// Delete 从缓存中删除值。
	// 如果找到并删除了键，则返回true；如果未找到键，则返回false。
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - key: The key to remove
	//
	// Returns:
	//   - bool: True if the key was found and removed
	//   - error: Error if the delete operation failed
	Delete(ctx context.Context, key string) (bool, error)

	// Clear removes all values from the cache.
	// This operation is atomic and thread-safe.
	//
	// Clear 删除缓存中的所有值。
	// 此操作是原子的且线程安全的。
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//
	// Returns:
	//   - error: Error if the clear operation failed
	Clear(ctx context.Context) error

	// Stats returns statistics about the cache.
	// This includes hit/miss counts, evictions, and memory usage.
	//
	// Stats 返回有关缓存的统计信息。
	// 这包括命中/未命中计数、淘汰次数和内存使用情况。
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//
	// Returns:
	//   - *Stats: Cache statistics
	//   - error: Error if retrieving statistics failed
	Stats(ctx context.Context) (*Stats, error)

	// Close cleans up resources used by the cache.
	// After calling Close, the cache should not be used anymore.
	//
	// Close 清理缓存使用的资源。
	// 调用Close后，不应再使用缓存。
	//
	// Returns:
	//   - error: Error if the close operation failed
	Close() error
}

// Stats represents cache statistics.
// These metrics are collected during cache operations and can be used
// to monitor performance and adjust cache parameters.
//
// Stats 表示缓存统计信息。
// 这些指标在缓存操作期间收集，可用于监控性能和调整缓存参数。
type Stats struct {
	// EntryCount is the current number of entries in the cache
	// EntryCount 是缓存中当前的条目数量
	EntryCount int64

	// Hits is the number of successful cache retrievals
	// Hits 是成功的缓存检索次数
	Hits int64

	// Misses is the number of cache retrievals where the key was not found
	// Misses 是未找到键的缓存检索次数
	Misses int64

	// Evictions is the number of entries removed due to capacity constraints
	// Evictions 是由于容量限制而删除的条目数
	Evictions int64

	// Size is the current memory usage of the cache in bytes
	// Size 是缓存当前的内存使用量（字节）
	Size int64
}
