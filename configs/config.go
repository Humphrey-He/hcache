// Package configs provides configuration structures and utilities for HCache.
// It offers mechanisms for loading, validating, and saving configuration from various sources
// including JSON and YAML files. The package defines a comprehensive configuration structure
// that controls all aspects of the cache system.
//
// Package configs 提供HCache的配置结构和工具。
// 它提供从各种来源（包括JSON和YAML文件）加载、验证和保存配置的机制。
// 该包定义了一个全面的配置结构，用于控制缓存系统的所有方面。
package configs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"

	"gopkg.in/yaml.v3"
)

// Config represents the complete configuration for HCache.
// It contains all settings needed to configure the cache system,
// organized into logical sections for different components.
//
// Config 表示HCache的完整配置。
// 它包含配置缓存系统所需的所有设置，
// 按不同组件的逻辑部分进行组织。
type Config struct {
	// Cache contains core cache settings like capacity and TTL
	// Cache 包含核心缓存设置，如容量和TTL
	Cache CacheConfig `json:"cache" yaml:"cache"`

	// Storage defines how cache items are stored and managed
	// Storage 定义缓存项如何存储和管理
	Storage StorageConfig `json:"storage" yaml:"storage"`

	// Admission controls which items are admitted to the cache
	// Admission 控制哪些项目被允许进入缓存
	Admission AdmissionConfig `json:"admission" yaml:"admission"`

	// Eviction defines how items are removed when the cache is full
	// Eviction 定义当缓存已满时如何移除项目
	Eviction EvictionConfig `json:"eviction" yaml:"eviction"`

	// Metrics configures performance monitoring and statistics
	// Metrics 配置性能监控和统计
	Metrics MetricsConfig `json:"metrics" yaml:"metrics"`

	// Log configures the logging behavior
	// Log 配置日志行为
	Log LogConfig `json:"log" yaml:"log"`

	// Extensions configures optional features like hot reloading
	// Extensions 配置可选功能，如热重载
	Extensions ExtensionsConfig `json:"extensions" yaml:"extensions"`

	// Extra allows for custom configuration options
	// Extra 允许自定义配置选项
	Extra map[string]interface{} `json:"extra" yaml:"extra"`
}

// CacheConfig contains settings for the cache itself.
// These settings control the core behavior of the cache,
// such as capacity limits and expiration policies.
//
// CacheConfig 包含缓存本身的设置。
// 这些设置控制缓存的核心行为，
// 如容量限制和过期策略。
type CacheConfig struct {
	// Enable determines whether the cache is active
	// Enable 确定缓存是否处于活动状态
	Enable bool `json:"enable" yaml:"enable"`

	// Name is the identifier for this cache instance
	// Name 是此缓存实例的标识符
	Name string `json:"name" yaml:"name"`

	// MaxEntries is the maximum number of items the cache can hold (0 = unlimited)
	// MaxEntries 是缓存可以容纳的最大项目数（0 = 无限制）
	MaxEntries int `json:"max_entries" yaml:"max_entries"`

	// MaxMemoryBytes is the maximum memory the cache can use in bytes (0 = unlimited)
	// MaxMemoryBytes 是缓存可以使用的最大内存（字节）（0 = 无限制）
	MaxMemoryBytes int64 `json:"max_memory_bytes" yaml:"max_memory_bytes"`

	// DefaultTTL is the default time-to-live for cache entries
	// DefaultTTL 是缓存条目的默认生存时间
	DefaultTTL time.Duration `json:"default_ttl" yaml:"default_ttl"`

	// CleanupInterval is how often expired items are removed
	// CleanupInterval 是清除过期项目的频率
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`
}

// StorageConfig contains settings for the storage backend.
// These settings control how cache items are physically stored
// and accessed, including optimization options.
//
// StorageConfig 包含存储后端的设置。
// 这些设置控制缓存项目如何物理存储和访问，
// 包括优化选项。
type StorageConfig struct {
	// Engine determines the storage implementation to use (e.g., "in-memory")
	// Engine 确定要使用的存储实现（例如，"in-memory"）
	Engine string `json:"engine" yaml:"engine"`

	// ShardCount is the number of shards for reducing lock contention (must be power of 2)
	// ShardCount 是用于减少锁竞争的分片数量（必须是2的幂）
	ShardCount int `json:"shard_count" yaml:"shard_count"`

	// EnableTTLTracking enables tracking of item expiration times
	// EnableTTLTracking 启用项目过期时间的跟踪
	EnableTTLTracking bool `json:"enable_ttl_tracking" yaml:"enable_ttl_tracking"`

	// EnableCompression enables compression of cache values to save memory
	// EnableCompression 启用缓存值的压缩以节省内存
	EnableCompression bool `json:"enable_compression" yaml:"enable_compression"`

	// CompressionThreshold is the minimum size in bytes for compression to be applied
	// CompressionThreshold 是应用压缩的最小大小（字节）
	CompressionThreshold int `json:"compression_threshold" yaml:"compression_threshold"`

	// EnableShardedLock enables fine-grained locking for better concurrency
	// EnableShardedLock 启用细粒度锁定以提高并发性
	EnableShardedLock bool `json:"enable_sharded_lock" yaml:"enable_sharded_lock"`
}

// AdmissionConfig contains settings for the admission policy.
// These settings control which items are allowed into the cache,
// helping to prevent cache pollution from infrequently accessed items.
//
// AdmissionConfig 包含准入策略的设置。
// 这些设置控制哪些项目被允许进入缓存，
// 帮助防止不常访问的项目污染缓存。
type AdmissionConfig struct {
	// Policy determines the admission algorithm (e.g., "frequency-sketch")
	// Policy 确定准入算法（例如，"frequency-sketch"）
	Policy string `json:"policy" yaml:"policy"`

	// SampleRate is the fraction of items to sample for admission decisions
	// SampleRate 是用于准入决策的采样项目比例
	SampleRate float64 `json:"sample_rate" yaml:"sample_rate"`

	// MinEntriesForAdmission is the threshold before admission control activates
	// MinEntriesForAdmission 是准入控制激活前的阈值
	MinEntriesForAdmission int `json:"min_entries_for_admission" yaml:"min_entries_for_admission"`

	// WindowSize is the size of the sliding window for frequency tracking
	// WindowSize 是频率跟踪的滑动窗口大小
	WindowSize int `json:"window_size" yaml:"window_size"`

	// Counters is the number of counters for frequency estimation
	// Counters 是频率估计的计数器数量
	Counters int `json:"counters" yaml:"counters"`
}

// EvictionConfig contains settings for the eviction policy.
// These settings control how items are selected for removal
// when the cache reaches capacity limits.
//
// EvictionConfig 包含淘汰策略的设置。
// 这些设置控制当缓存达到容量限制时如何选择要移除的项目。
type EvictionConfig struct {
	// Policy determines the eviction algorithm (e.g., "lru", "lfu", "fifo", "random")
	// Policy 确定淘汰算法（例如，"lru"、"lfu"、"fifo"、"random"）
	Policy string `json:"policy" yaml:"policy"`

	// BatchSize is the number of items to consider in each eviction round
	// BatchSize 是每轮淘汰中要考虑的项目数量
	BatchSize int `json:"batch_size" yaml:"batch_size"`

	// SampleRatio is the fraction of cache to sample for eviction candidates
	// SampleRatio 是用于淘汰候选的缓存采样比例
	SampleRatio float64 `json:"sample_ratio" yaml:"sample_ratio"`

	// MinTTLSeconds is the minimum remaining TTL to protect items from eviction
	// MinTTLSeconds 是保护项目不被淘汰的最小剩余TTL（秒）
	MinTTLSeconds int `json:"min_ttl_seconds" yaml:"min_ttl_seconds"`

	// MaxEvictionRatio is the maximum fraction of cache to evict at once
	// MaxEvictionRatio 是一次淘汰的最大缓存比例
	MaxEvictionRatio float64 `json:"max_eviction_ratio" yaml:"max_eviction_ratio"`
}

// MetricsConfig contains settings for metrics collection.
// These settings control how performance data is collected,
// processed, and exposed for monitoring.
//
// MetricsConfig 包含指标收集的设置。
// 这些设置控制如何收集、处理和暴露性能数据以进行监控。
type MetricsConfig struct {
	// Enable determines whether metrics collection is active
	// Enable 确定是否启用指标收集
	Enable bool `json:"enable" yaml:"enable"`

	// Level controls the detail of metrics collection ("basic", "detailed", "disabled")
	// Level 控制指标收集的详细程度（"basic"、"detailed"、"disabled"）
	Level string `json:"level" yaml:"level"`

	// PrometheusPort is the port for exposing Prometheus metrics
	// PrometheusPort 是暴露Prometheus指标的端口
	PrometheusPort int `json:"prometheus_port" yaml:"prometheus_port"`

	// ExportInterval is how often metrics are aggregated and exported
	// ExportInterval 是指标聚合和导出的频率
	ExportInterval time.Duration `json:"export_interval" yaml:"export_interval"`

	// HistogramBuckets defines latency histogram buckets in milliseconds
	// HistogramBuckets 定义延迟直方图桶（毫秒）
	HistogramBuckets []float64 `json:"histogram_buckets" yaml:"histogram_buckets"`
}

// LogConfig contains settings for logging.
// These settings control the logging behavior, including
// log level, format, and output destination.
//
// LogConfig 包含日志记录的设置。
// 这些设置控制日志行为，包括日志级别、格式和输出目的地。
type LogConfig struct {
	// Level sets the minimum log level ("debug", "info", "warn", "error")
	// Level 设置最低日志级别（"debug"、"info"、"warn"、"error"）
	Level string `json:"level" yaml:"level"`

	// Format specifies the log format ("text", "json")
	// Format 指定日志格式（"text"、"json"）
	Format string `json:"format" yaml:"format"`

	// Output determines where logs are written ("stdout", "stderr", "file")
	// Output 确定日志写入的位置（"stdout"、"stderr"、"file"）
	Output string `json:"output" yaml:"output"`

	// FilePath is the path to the log file when Output is "file"
	// FilePath 是当Output为"file"时的日志文件路径
	FilePath string `json:"file_path" yaml:"file_path"`

	// MaxSizeMB is the maximum log file size before rotation
	// MaxSizeMB 是轮换前的最大日志文件大小（MB）
	MaxSizeMB int `json:"max_size_mb" yaml:"max_size_mb"`

	// MaxBackups is the number of rotated log files to keep
	// MaxBackups 是要保留的轮换日志文件数量
	MaxBackups int `json:"max_backups" yaml:"max_backups"`

	// MaxAgeDays is the maximum age of log files in days
	// MaxAgeDays 是日志文件的最大保留天数
	MaxAgeDays int `json:"max_age_days" yaml:"max_age_days"`
}

// ExtensionsConfig contains settings for extensions.
// These settings control optional features that extend
// the core functionality of the cache.
//
// ExtensionsConfig 包含扩展的设置。
// 这些设置控制扩展缓存核心功能的可选功能。
type ExtensionsConfig struct {
	// HotReload contains settings for dynamic configuration reloading
	// HotReload 包含动态配置重新加载的设置
	HotReload HotReloadConfig `json:"hot_reload" yaml:"hot_reload"`
}

// HotReloadConfig contains settings for hot reloading.
// These settings control how configuration changes are
// detected and applied without system restart.
//
// HotReloadConfig 包含热重载的设置。
// 这些设置控制如何检测和应用配置更改而无需重启系统。
type HotReloadConfig struct {
	// Enable determines whether hot reloading is active
	// Enable 确定是否启用热重载
	Enable bool `json:"enable" yaml:"enable"`

	// WatchInterval is how often to check for configuration changes
	// WatchInterval 是检查配置更改的频率
	WatchInterval time.Duration `json:"watch_interval" yaml:"watch_interval"`
}

// DefaultConfig returns a new Config with default values.
// This provides a starting point for configuration with reasonable defaults
// for all settings, which can then be customized as needed.
//
// DefaultConfig 返回具有默认值的新Config。
// 这为所有设置提供了具有合理默认值的配置起点，
// 然后可以根据需要进行自定义。
//
// Returns:
//   - *Config: A new configuration instance with default values
//
// 返回：
//   - *Config: 具有默认值的新配置实例
func DefaultConfig() *Config {
	return &Config{
		Cache: CacheConfig{
			Enable:          true,
			Name:            "hcache",
			MaxEntries:      500000,
			MaxMemoryBytes:  1073741824, // 1GB
			DefaultTTL:      300 * time.Second,
			CleanupInterval: 30 * time.Second,
		},
		Storage: StorageConfig{
			Engine:               "in-memory",
			ShardCount:           256,
			EnableTTLTracking:    true,
			EnableCompression:    false,
			CompressionThreshold: 4096,
			EnableShardedLock:    true,
		},
		Admission: AdmissionConfig{
			Policy:                 "frequency-sketch",
			SampleRate:             0.01,
			MinEntriesForAdmission: 100,
			WindowSize:             10000,
			Counters:               4,
		},
		Eviction: EvictionConfig{
			Policy:           "lfu",
			BatchSize:        128,
			SampleRatio:      0.1,
			MinTTLSeconds:    10,
			MaxEvictionRatio: 0.25,
		},
		Metrics: MetricsConfig{
			Enable:           true,
			Level:            "basic",
			PrometheusPort:   2112,
			ExportInterval:   10 * time.Second,
			HistogramBuckets: []float64{0.1, 0.5, 1, 5, 10, 50, 100, 500},
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			FilePath:   "/var/log/hcache.log",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 28,
		},
		Extensions: ExtensionsConfig{
			HotReload: HotReloadConfig{
				Enable:        false,
				WatchInterval: 30 * time.Second,
			},
		},
		Extra: make(map[string]interface{}),
	}
}

// LoadFromFile loads configuration from a file.
// It supports both YAML and JSON formats, automatically
// detecting the format based on the file extension.
//
// LoadFromFile 从文件加载配置。
// 它支持YAML和JSON格式，根据文件扩展名自动检测格式。
//
// Parameters:
//   - filename: Path to the configuration file
//
// Returns:
//   - *Config: The loaded configuration
//   - error: An error if loading fails
//
// 参数：
//   - filename: 配置文件的路径
//
// 返回：
//   - *Config: 加载的配置
//   - error: 如果加载失败则返回错误
func LoadFromFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	config := DefaultConfig()
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".yaml", ".yml":
		err = yaml.NewDecoder(file).Decode(config)
	case ".json":
		err = json.NewDecoder(file).Decode(config)
	default:
		return nil, fmt.Errorf("unsupported configuration file format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return config, nil
}

// LoadFromReader loads configuration from an io.Reader.
// This allows loading configuration from sources other than files,
// such as network streams or in-memory data.
//
// LoadFromReader 从io.Reader加载配置。
// 这允许从文件以外的源加载配置，
// 如网络流或内存中的数据。
//
// Parameters:
//   - r: The reader providing the configuration data
//   - format: The format of the data ("json", "yaml", or "yml")
//
// Returns:
//   - *Config: The loaded configuration
//   - error: An error if loading fails
//
// 参数：
//   - r: 提供配置数据的读取器
//   - format: 数据的格式（"json"、"yaml"或"yml"）
//
// 返回：
//   - *Config: 加载的配置
//   - error: 如果加载失败则返回错误
func LoadFromReader(r io.Reader, format string) (*Config, error) {
	config := DefaultConfig()
	var err error

	switch strings.ToLower(format) {
	case "yaml", "yml":
		err = yaml.NewDecoder(r).Decode(config)
	case "json":
		err = json.NewDecoder(r).Decode(config)
	default:
		return nil, fmt.Errorf("unsupported configuration format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return config, nil
}

// SaveToFile saves configuration to a file.
// It supports both YAML and JSON formats, automatically
// selecting the format based on the file extension.
//
// SaveToFile 将配置保存到文件。
// 它支持YAML和JSON格式，根据文件扩展名自动选择格式。
//
// Parameters:
//   - filename: Path where the configuration will be saved
//
// Returns:
//   - error: An error if saving fails
//
// 参数：
//   - filename: 配置将保存的路径
//
// 返回：
//   - error: 如果保存失败则返回错误
func (c *Config) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create configuration file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".yaml", ".yml":
		encoder := yaml.NewEncoder(file)
		defer encoder.Close()
		err = encoder.Encode(c)
	case ".json":
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(c)
	default:
		return fmt.Errorf("unsupported configuration file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to encode configuration: %w", err)
	}

	return nil
}

// Validate validates the configuration.
// It checks that all settings have valid values and
// that there are no conflicts or inconsistencies.
//
// Validate 验证配置。
// 它检查所有设置是否具有有效值，
// 并且没有冲突或不一致。
//
// Returns:
//   - error: An error describing the validation failure, or nil if valid
//
// 返回：
//   - error: 描述验证失败的错误，如果有效则为nil
func (c *Config) Validate() error {
	// Validate cache settings
	// 验证缓存设置
	if c.Cache.MaxEntries < 0 {
		return fmt.Errorf("cache.max_entries must be non-negative")
	}
	if c.Cache.MaxMemoryBytes < 0 {
		return fmt.Errorf("cache.max_memory_bytes must be non-negative")
	}
	if c.Cache.CleanupInterval < time.Second {
		return fmt.Errorf("cache.cleanup_interval must be at least 1 second")
	}

	// Validate storage settings
	// 验证存储设置
	if c.Storage.ShardCount <= 0 {
		return fmt.Errorf("storage.shard_count must be positive")
	}
	if !isPowerOfTwo(c.Storage.ShardCount) {
		return fmt.Errorf("storage.shard_count must be a power of 2")
	}
	if c.Storage.CompressionThreshold < 0 {
		return fmt.Errorf("storage.compression_threshold must be non-negative")
	}

	// Validate admission settings
	// 验证准入设置
	if c.Admission.SampleRate < 0 || c.Admission.SampleRate > 1 {
		return fmt.Errorf("admission.sample_rate must be between 0 and 1")
	}
	if c.Admission.MinEntriesForAdmission < 0 {
		return fmt.Errorf("admission.min_entries_for_admission must be non-negative")
	}
	if c.Admission.WindowSize <= 0 {
		return fmt.Errorf("admission.window_size must be positive")
	}
	if c.Admission.Counters <= 0 {
		return fmt.Errorf("admission.counters must be positive")
	}

	// Validate eviction settings
	// 验证淘汰设置
	switch c.Eviction.Policy {
	case "lru", "lfu", "fifo", "random":
		// Valid policies
		// 有效策略
	default:
		return fmt.Errorf("eviction.policy must be one of: lru, lfu, fifo, random")
	}
	if c.Eviction.BatchSize <= 0 {
		return fmt.Errorf("eviction.batch_size must be positive")
	}
	if c.Eviction.SampleRatio <= 0 || c.Eviction.SampleRatio > 1 {
		return fmt.Errorf("eviction.sample_ratio must be between 0 and 1")
	}
	if c.Eviction.MinTTLSeconds < 0 {
		return fmt.Errorf("eviction.min_ttl_seconds must be non-negative")
	}
	if c.Eviction.MaxEvictionRatio <= 0 || c.Eviction.MaxEvictionRatio > 1 {
		return fmt.Errorf("eviction.max_eviction_ratio must be between 0 and 1")
	}

	// Validate metrics settings
	// 验证指标设置
	if c.Metrics.Enable {
		if c.Metrics.PrometheusPort <= 0 || c.Metrics.PrometheusPort > 65535 {
			return fmt.Errorf("metrics.prometheus_port must be between 1 and 65535")
		}
		if c.Metrics.ExportInterval < time.Second {
			return fmt.Errorf("metrics.export_interval must be at least 1 second")
		}
	}

	// Validate log settings
	// 验证日志设置
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
		// Valid levels
		// 有效级别
	default:
		return fmt.Errorf("log.level must be one of: debug, info, warn, error")
	}
	switch c.Log.Format {
	case "text", "json":
		// Valid formats
		// 有效格式
	default:
		return fmt.Errorf("log.format must be one of: text, json")
	}
	switch c.Log.Output {
	case "stdout", "stderr", "file":
		// Valid outputs
		// 有效输出
	default:
		return fmt.Errorf("log.output must be one of: stdout, stderr, file")
	}
	if c.Log.Output == "file" && c.Log.FilePath == "" {
		return fmt.Errorf("log.file_path must be specified when log.output is 'file'")
	}
	if c.Log.MaxSizeMB <= 0 {
		return fmt.Errorf("log.max_size_mb must be positive")
	}
	if c.Log.MaxBackups < 0 {
		return fmt.Errorf("log.max_backups must be non-negative")
	}
	if c.Log.MaxAgeDays < 0 {
		return fmt.Errorf("log.max_age_days must be non-negative")
	}

	// Validate extensions settings
	// 验证扩展设置
	if c.Extensions.HotReload.Enable && c.Extensions.HotReload.WatchInterval < time.Second {
		return fmt.Errorf("extensions.hot_reload.watch_interval must be at least 1 second")
	}

	return nil
}

// isPowerOfTwo checks if n is a power of 2.
// This is used to validate that shard counts are powers of 2,
// which is important for efficient hashing.
//
// isPowerOfTwo 检查n是否为2的幂。
// 这用于验证分片计数是否为2的幂，
// 这对于高效哈希很重要。
//
// Parameters:
//   - n: The number to check
//
// Returns:
//   - bool: True if n is a power of 2, false otherwise
//
// 参数：
//   - n: 要检查的数字
//
// 返回：
//   - bool: 如果n是2的幂则为true，否则为false
func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1)) == 0
}
