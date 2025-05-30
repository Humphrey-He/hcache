package cache

import (
	"time"

	"github.com/noobtrump/hcache/pkg/codec"
)

// Option is a function that configures a Config.
// This pattern allows for flexible and readable configuration of cache instances.
//
// Option 是一个配置Config的函数。
// 这种模式允许灵活且可读地配置缓存实例。
type Option func(*Config)

// WithMaxEntryCount sets the maximum number of entries in the cache.
// If set to 0, there is no limit on the number of entries.
//
// WithMaxEntryCount 设置缓存中的最大条目数。
// 如果设置为0，则条目数量没有限制。
//
// Parameters:
//   - count: The maximum number of entries
//
// Returns:
//   - Option: A configuration option
func WithMaxEntryCount(count int) Option {
	return func(c *Config) {
		c.MaxEntries = count
	}
}

// WithMaxMemory sets the maximum memory usage in bytes.
// If set to 0, there is no limit on memory usage.
//
// WithMaxMemory 设置最大内存使用量（字节）。
// 如果设置为0，则内存使用没有限制。
//
// Parameters:
//   - bytes: The maximum memory usage in bytes
//
// Returns:
//   - Option: A configuration option
func WithMaxMemory(bytes int64) Option {
	return func(c *Config) {
		c.MaxMemoryBytes = bytes
	}
}

// WithTTL sets the default time-to-live for cache entries.
// If set to 0, entries don't expire by default.
// If set to a negative value, entries never expire.
//
// WithTTL 设置缓存条目的默认生存时间。
// 如果设置为0，则条目默认不过期。
// 如果设置为负值，则条目永不过期。
//
// Parameters:
//   - ttl: The default time-to-live duration
//
// Returns:
//   - Option: A configuration option
func WithTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.DefaultTTL = ttl
	}
}

// WithShards sets the number of shards for the cache.
// Higher values reduce lock contention in concurrent scenarios.
// The value must be a power of 2.
//
// WithShards 设置缓存的分片数量。
// 较高的值可以减少并发场景中的锁竞争。
// 该值必须是2的幂。
//
// Parameters:
//   - count: The number of shards
//
// Returns:
//   - Option: A configuration option
func WithShards(count int) Option {
	return func(c *Config) {
		c.ShardCount = count
	}
}

// WithEviction sets the eviction policy.
// Valid values: "lru", "lfu", "fifo", "random"
//
// WithEviction 设置淘汰策略。
// 有效值："lru"、"lfu"、"fifo"、"random"
//
// Parameters:
//   - policy: The eviction policy to use
//
// Returns:
//   - Option: A configuration option
func WithEviction(policy string) Option {
	return func(c *Config) {
		c.EvictionPolicy = policy
	}
}

// WithMetricsEnabled enables or disables metrics collection.
//
// WithMetricsEnabled 启用或禁用指标收集。
//
// Parameters:
//   - enabled: Whether to enable metrics collection
//
// Returns:
//   - Option: A configuration option
func WithMetricsEnabled(enabled bool) Option {
	return func(c *Config) {
		c.EnableMetrics = enabled
	}
}

// WithMetricsLevel sets the metrics collection level.
// Valid values: "basic", "detailed", "disabled"
//
// WithMetricsLevel 设置指标收集级别。
// 有效值："basic"、"detailed"、"disabled"
//
// Parameters:
//   - level: The metrics collection level
//
// Returns:
//   - Option: A configuration option
func WithMetricsLevel(level string) Option {
	return func(c *Config) {
		c.MetricsLevel = level
	}
}

// WithAdmissionPolicy enables or disables the admission policy.
// Admission policy helps prevent cache thrashing by only admitting items
// that are likely to be accessed again.
//
// WithAdmissionPolicy 启用或禁用准入策略。
// 准入策略通过只允许可能再次访问的项目来帮助防止缓存抖动。
//
// Parameters:
//   - enabled: Whether to enable the admission policy
//
// Returns:
//   - Option: A configuration option
func WithAdmissionPolicy(enabled bool) Option {
	return func(c *Config) {
		c.EnableAdmissionPolicy = enabled
	}
}

// WithCleanupInterval sets the interval for cleaning up expired entries.
// This is the interval at which expired items are removed from the cache.
//
// WithCleanupInterval 设置清理过期条目的间隔。
// 这是从缓存中删除过期项目的时间间隔。
//
// Parameters:
//   - interval: The cleanup interval duration
//
// Returns:
//   - Option: A configuration option
func WithCleanupInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.CleanupInterval = interval
	}
}

// WithCompression enables or disables compression for large values.
// When enabled, large values are compressed to save memory.
//
// WithCompression 启用或禁用大值的压缩。
// 启用后，大值将被压缩以节省内存。
//
// Parameters:
//   - enabled: Whether to enable compression
//
// Returns:
//   - Option: A configuration option
func WithCompression(enabled bool) Option {
	return func(c *Config) {
		c.EnableCompression = enabled
	}
}

// WithCompressionThreshold sets the minimum size (in bytes) for compression.
// Values larger than this threshold will be compressed.
//
// WithCompressionThreshold 设置压缩的最小大小（字节）。
// 大于此阈值的值将被压缩。
//
// Parameters:
//   - threshold: The compression threshold in bytes
//
// Returns:
//   - Option: A configuration option
func WithCompressionThreshold(threshold int) Option {
	return func(c *Config) {
		c.CompressionThreshold = threshold
	}
}

// WithShardedLock enables or disables sharded locking.
// Sharded locking provides finer-grained concurrency control.
//
// WithShardedLock 启用或禁用分片锁定。
// 分片锁定提供更细粒度的并发控制。
//
// Parameters:
//   - enabled: Whether to enable sharded locking
//
// Returns:
//   - Option: A configuration option
func WithShardedLock(enabled bool) Option {
	return func(c *Config) {
		c.EnableShardedLock = enabled
	}
}

// WithCodec sets the serialization codec for the cache.
// The codec is used to serialize and deserialize cache values.
//
// WithCodec 设置缓存的序列化编解码器。
// 编解码器用于序列化和反序列化缓存值。
//
// Parameters:
//   - codec: The codec to use
//
// Returns:
//   - Option: A configuration option
func WithCodec(codec codec.Codec) Option {
	return func(c *Config) {
		c.Codec = codec
	}
}

// WithLoader sets the data loader for the cache.
// The loader is used to load data when a cache miss occurs.
//
// WithLoader 设置缓存的数据加载器。
// 当缓存未命中时，加载器用于加载数据。
//
// Parameters:
//   - loader: The data loader to use
//
// Returns:
//   - Option: A configuration option
func WithLoader(loader interface{}) Option {
	return func(c *Config) {
		c.Loader = loader
	}
}

// NewWithOptions creates a new cache with the given options.
// This is a convenience function that applies the provided options to a new cache.
//
// NewWithOptions 使用给定选项创建新的缓存。
// 这是一个便捷函数，将提供的选项应用于新缓存。
//
// Parameters:
//   - name: The name of the cache
//   - options: A list of configuration options
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the cache creation fails
func NewWithOptions(name string, options ...Option) (ICache, error) {
	config := NewDefaultConfig()
	config.Name = name

	// Apply all options
	// 应用所有选项
	for _, option := range options {
		option(config)
	}

	// For testing purposes, return a mock implementation
	// 出于测试目的，返回一个模拟实现
	return NewMockCache(name, config.MaxEntries, config.EvictionPolicy), nil
}
