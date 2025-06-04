package core

import "time"

// Option is a function type for configuring a cache instance.
// It follows the functional options pattern, allowing for flexible and
// readable configuration.
type Option func(config *Config)

// Config holds the configuration parameters for a cache instance.
type Config struct {
	// Name is a unique identifier for the cache instance
	Name string

	// MaxEntryCount is the maximum number of entries the cache can hold
	// When this limit is reached, the eviction policy is used to remove entries
	MaxEntryCount int64

	// MaxMemoryBytes is the maximum memory usage in bytes
	// When this limit is reached, the eviction policy is used to remove entries
	MaxMemoryBytes int64

	// DefaultTTL is the default time-to-live for cache entries
	// Entries will be automatically removed after this duration unless a custom TTL is specified
	DefaultTTL time.Duration

	// EvictionPolicy determines how entries are selected for removal when the cache is full
	// Valid values are "lru", "lfu", "fifo", and "random"
	EvictionPolicy string

	// Shards is the number of segments to divide the cache into for better concurrency
	// Higher values reduce lock contention but increase memory overhead
	Shards int

	// MetricsEnabled determines whether performance metrics are collected
	MetricsEnabled bool

	// CleanupInterval is how often the cache checks for and removes expired entries
	CleanupInterval time.Duration
}

// WithMaxEntryCount sets the maximum number of entries the cache can hold.
func WithMaxEntryCount(count int64) Option {
	return func(config *Config) {
		config.MaxEntryCount = count
	}
}

// WithMaxMemoryBytes sets the maximum memory usage in bytes.
func WithMaxMemoryBytes(bytes int64) Option {
	return func(config *Config) {
		config.MaxMemoryBytes = bytes
	}
}

// WithTTL sets the default time-to-live for cache entries.
func WithTTL(ttl time.Duration) Option {
	return func(config *Config) {
		config.DefaultTTL = ttl
	}
}

// WithEvictionPolicy sets the policy for evicting entries when the cache is full.
// Valid values are "lru", "lfu", "fifo", and "random".
func WithEvictionPolicy(policy string) Option {
	return func(config *Config) {
		config.EvictionPolicy = policy
	}
}

// WithShards sets the number of segments to divide the cache into.
func WithShards(shards int) Option {
	return func(config *Config) {
		config.Shards = shards
	}
}

// WithMetricsEnabled enables or disables performance metrics collection.
func WithMetricsEnabled(enabled bool) Option {
	return func(config *Config) {
		config.MetricsEnabled = enabled
	}
}

// WithCleanupInterval sets how often the cache checks for and removes expired entries.
func WithCleanupInterval(interval time.Duration) Option {
	return func(config *Config) {
		config.CleanupInterval = interval
	}
}
