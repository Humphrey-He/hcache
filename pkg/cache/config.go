package cache

import (
	"fmt"
	"time"

	"github.com/yourusername/hcache/pkg/codec"
)

// Config defines the configuration options for a cache instance.
// It controls behavior such as capacity limits, eviction policies, and performance options.
//
// Config 定义缓存实例的配置选项。
// 它控制诸如容量限制、淘汰策略和性能选项等行为。
type Config struct {
	// Name of the cache instance, used for metrics and logging
	// 缓存实例的名称，用于指标收集和日志记录
	Name string `json:"name" yaml:"name"`

	// MaxEntries is the maximum number of entries the cache can hold
	// If set to 0, there is no limit on the number of entries
	//
	// MaxEntries 是缓存可以容纳的最大条目数
	// 如果设置为0，则条目数量没有限制
	MaxEntries int `json:"max_entries" yaml:"max_entries"`

	// MaxMemoryBytes is the maximum memory the cache can use (in bytes)
	// If set to 0, there is no limit on memory usage
	//
	// MaxMemoryBytes 是缓存可以使用的最大内存（字节）
	// 如果设置为0，则内存使用没有限制
	MaxMemoryBytes int64 `json:"max_memory_bytes" yaml:"max_memory_bytes"`

	// DefaultTTL is the default time-to-live for cache entries
	// If set to 0, entries don't expire by default
	//
	// DefaultTTL 是缓存条目的默认生存时间
	// 如果设置为0，则条目默认不过期
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl"`

	// ShardCount is the number of shards to use for the cache
	// Higher values reduce lock contention in concurrent scenarios
	//
	// ShardCount 是缓存使用的分片数量
	// 较高的值可以减少并发场景中的锁竞争
	ShardCount int `json:"shard_count" yaml:"shard_count"`

	// EvictionPolicy determines which items to evict when the cache is full
	// Valid values: "lru", "lfu", "fifo", "random"
	//
	// EvictionPolicy 决定当缓存已满时要淘汰哪些项目
	// 有效值："lru"、"lfu"、"fifo"、"random"
	EvictionPolicy string `json:"eviction_policy" yaml:"eviction_policy"`

	// EnableMetrics determines whether to collect performance metrics
	//
	// EnableMetrics 决定是否收集性能指标
	EnableMetrics bool `json:"enable_metrics" yaml:"enable_metrics"`

	// MetricsLevel controls the detail level of metrics collection
	// Valid values: "basic", "detailed", "disabled"
	//
	// MetricsLevel 控制指标收集的详细程度
	// 有效值："basic"、"detailed"、"disabled"
	MetricsLevel string `json:"metrics_level" yaml:"metrics_level"`

	// EnableAdmissionPolicy enables the admission policy to prevent cache thrashing
	// This helps protect against cache pollution from infrequently accessed items
	//
	// EnableAdmissionPolicy 启用准入策略以防止缓存抖动
	// 这有助于防止不常访问的项目污染缓存
	EnableAdmissionPolicy bool `json:"enable_admission_policy" yaml:"enable_admission_policy"`

	// CleanupInterval is the interval at which expired items are cleaned up
	//
	// CleanupInterval 是清理过期项目的时间间隔
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`

	// EnableCompression enables compression for large values to save memory
	//
	// EnableCompression 启用大值的压缩以节省内存
	EnableCompression bool `json:"enable_compression" yaml:"enable_compression"`

	// CompressionThreshold is the minimum size (in bytes) for a value to be compressed
	//
	// CompressionThreshold 是值被压缩的最小大小（字节）
	CompressionThreshold int `json:"compression_threshold" yaml:"compression_threshold"`

	// EnableShardedLock enables fine-grained locking for better concurrency
	//
	// EnableShardedLock 启用细粒度锁定以提高并发性
	EnableShardedLock bool `json:"enable_sharded_lock" yaml:"enable_sharded_lock"`

	// Codec is the serialization codec to use for storing values
	// If nil, the default JSON codec will be used
	//
	// Codec 是用于存储值的序列化编解码器
	// 如果为nil，将使用默认的JSON编解码器
	Codec codec.Codec `json:"-" yaml:"-"`
}

// NewDefaultConfig returns a Config with sensible default values.
// This provides a starting point for creating a cache configuration.
//
// NewDefaultConfig 返回具有合理默认值的Config。
// 这为创建缓存配置提供了一个起点。
//
// Returns:
//   - *Config: A new configuration instance with default values
func NewDefaultConfig() *Config {
	return &Config{
		Name:                  "hcache",
		MaxEntries:            10000,
		MaxMemoryBytes:        100 * 1024 * 1024, // 100 MB
		DefaultTTL:            time.Hour,
		ShardCount:            16,
		EvictionPolicy:        "lru",
		EnableMetrics:         true,
		MetricsLevel:          "basic",
		EnableAdmissionPolicy: false,
		CleanupInterval:       time.Minute,
		EnableCompression:     false,
		CompressionThreshold:  4096, // 4 KB
		EnableShardedLock:     true,
		Codec:                 codec.DefaultCodec(),
	}
}

// Validate checks if the configuration is valid.
// It verifies that all settings have appropriate values and combinations.
//
// Validate 检查配置是否有效。
// 它验证所有设置是否具有适当的值和组合。
//
// Returns:
//   - error: An error if the configuration is invalid, nil otherwise
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("cache name cannot be empty")
	}

	if c.ShardCount <= 0 {
		return fmt.Errorf("shard count must be positive")
	}

	// Check if ShardCount is a power of 2
	// 检查ShardCount是否为2的幂
	if (c.ShardCount & (c.ShardCount - 1)) != 0 {
		return fmt.Errorf("shard count must be a power of 2")
	}

	// Validate eviction policy
	// 验证淘汰策略
	switch c.EvictionPolicy {
	case "lru", "lfu", "fifo", "random":
		// Valid policies
		// 有效策略
	default:
		return fmt.Errorf("invalid eviction policy: %s", c.EvictionPolicy)
	}

	// Validate metrics level
	// 验证指标级别
	switch c.MetricsLevel {
	case "basic", "detailed", "disabled":
		// Valid levels
		// 有效级别
	default:
		return fmt.Errorf("invalid metrics level: %s", c.MetricsLevel)
	}

	if c.CleanupInterval < time.Second {
		return fmt.Errorf("cleanup interval must be at least 1 second")
	}

	if c.EnableCompression && c.CompressionThreshold < 64 {
		return fmt.Errorf("compression threshold must be at least 64 bytes")
	}

	return nil
}

// WithName sets the cache name.
// The name is used for metrics and logging.
//
// WithName 设置缓存名称。
// 名称用于指标和日志记录。
//
// Parameters:
//   - name: The name to set
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithName(name string) *Config {
	c.Name = name
	return c
}

// WithMaxEntries sets the maximum number of entries.
// If set to 0, there is no limit on the number of entries.
//
// WithMaxEntries 设置最大条目数。
// 如果设置为0，则条目数量没有限制。
//
// Parameters:
//   - max: The maximum number of entries
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithMaxEntries(max int) *Config {
	c.MaxEntries = max
	return c
}

// WithMaxMemory sets the maximum memory usage in bytes.
// If set to 0, there is no limit on memory usage.
//
// WithMaxMemory 设置最大内存使用量（字节）。
// 如果设置为0，则内存使用没有限制。
//
// Parameters:
//   - maxBytes: The maximum memory usage in bytes
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithMaxMemory(maxBytes int64) *Config {
	c.MaxMemoryBytes = maxBytes
	return c
}

// WithDefaultTTL sets the default time-to-live for cache entries.
// If set to 0, entries don't expire by default.
//
// WithDefaultTTL 设置缓存条目的默认生存时间。
// 如果设置为0，则条目默认不过期。
//
// Parameters:
//   - ttl: The default time-to-live duration
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithDefaultTTL(ttl time.Duration) *Config {
	c.DefaultTTL = ttl
	return c
}

// WithShardCount sets the number of shards.
// Higher values reduce lock contention in concurrent scenarios.
// The value must be a power of 2.
//
// WithShardCount 设置分片数量。
// 较高的值可以减少并发场景中的锁竞争。
// 该值必须是2的幂。
//
// Parameters:
//   - count: The number of shards
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithShardCount(count int) *Config {
	c.ShardCount = count
	return c
}

// WithEvictionPolicy sets the eviction policy.
// Valid values: "lru", "lfu", "fifo", "random".
//
// WithEvictionPolicy 设置淘汰策略。
// 有效值："lru"、"lfu"、"fifo"、"random"。
//
// Parameters:
//   - policy: The eviction policy to use
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithEvictionPolicy(policy string) *Config {
	c.EvictionPolicy = policy
	return c
}

// WithMetrics enables or disables metrics collection.
//
// WithMetrics 启用或禁用指标收集。
//
// Parameters:
//   - enable: Whether to enable metrics collection
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithMetrics(enable bool) *Config {
	c.EnableMetrics = enable
	return c
}

// WithMetricsLevel sets the metrics collection level.
// Valid values: "basic", "detailed", "disabled".
//
// WithMetricsLevel 设置指标收集级别。
// 有效值："basic"、"detailed"、"disabled"。
//
// Parameters:
//   - level: The metrics collection level
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithMetricsLevel(level string) *Config {
	c.MetricsLevel = level
	return c
}

// WithAdmissionPolicy enables or disables the admission policy.
// Admission policy helps prevent cache thrashing by only admitting items
// that are likely to be accessed again.
//
// WithAdmissionPolicy 启用或禁用准入策略。
// 准入策略通过只允许可能再次访问的项目来帮助防止缓存抖动。
//
// Parameters:
//   - enable: Whether to enable the admission policy
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithAdmissionPolicy(enable bool) *Config {
	c.EnableAdmissionPolicy = enable
	return c
}

// WithCleanupInterval sets the cleanup interval.
// This is the interval at which expired items are removed from the cache.
//
// WithCleanupInterval 设置清理间隔。
// 这是从缓存中删除过期项目的时间间隔。
//
// Parameters:
//   - interval: The cleanup interval duration
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithCleanupInterval(interval time.Duration) *Config {
	c.CleanupInterval = interval
	return c
}

// WithCompression enables or disables compression.
// When enabled, large values are compressed to save memory.
//
// WithCompression 启用或禁用压缩。
// 启用后，大值将被压缩以节省内存。
//
// Parameters:
//   - enable: Whether to enable compression
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithCompression(enable bool) *Config {
	c.EnableCompression = enable
	return c
}

// WithCompressionThreshold sets the compression threshold.
// Values larger than this threshold (in bytes) will be compressed.
//
// WithCompressionThreshold 设置压缩阈值。
// 大于此阈值（字节）的值将被压缩。
//
// Parameters:
//   - threshold: The compression threshold in bytes
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithCompressionThreshold(threshold int) *Config {
	c.CompressionThreshold = threshold
	return c
}

// WithShardedLock enables or disables sharded locking.
// Sharded locking provides finer-grained concurrency control.
//
// WithShardedLock 启用或禁用分片锁定。
// 分片锁定提供更细粒度的并发控制。
//
// Parameters:
//   - enable: Whether to enable sharded locking
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithShardedLock(enable bool) *Config {
	c.EnableShardedLock = enable
	return c
}

// WithCodec sets the serialization codec.
// The codec is used to serialize and deserialize cache values.
//
// WithCodec 设置序列化编解码器。
// 编解码器用于序列化和反序列化缓存值。
//
// Parameters:
//   - codec: The codec to use
//
// Returns:
//   - *Config: The modified configuration (for method chaining)
func (c *Config) WithCodec(codec codec.Codec) *Config {
	c.Codec = codec
	return c
}
