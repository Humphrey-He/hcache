# HCache Configuration

This directory contains configuration files and utilities for HCache.

## Configuration Files

- `config.yaml`: The main configuration file in YAML format
- `config.json`: The main configuration file in JSON format

## Configuration Structure

The configuration is divided into several sections:

### Cache

Basic cache settings:
- `enable`: Whether to enable caching
- `name`: Name of the cache instance (used in metrics and logs)
- `max_entries`: Maximum number of entries in the cache
- `max_memory_bytes`: Maximum memory usage in bytes
- `default_ttl`: Default time-to-live for cache entries
- `cleanup_interval`: Interval for cleaning expired entries

### Storage

Storage backend settings:
- `engine`: Storage backend (in-memory, file, redis)
- `shard_count`: Number of shards for better concurrency
- `enable_ttl_tracking`: Whether to track TTL for entries
- `enable_compression`: Whether to compress large values
- `compression_threshold`: Minimum size in bytes for compression
- `enable_sharded_lock`: Whether to use fine-grained locking

### Admission

Admission policy settings:
- `policy`: Admission policy (none, count-min, frequency-sketch)
- `sample_rate`: Sampling rate for admission policy
- `min_entries_for_admission`: Minimum entries before admission control activates
- `window_size`: Window size for frequency counting
- `counters`: Number of counters for frequency sketch

### Eviction

Eviction policy settings:
- `policy`: Eviction policy (lru, lfu, fifo, random)
- `batch_size`: Number of entries to check in each eviction round
- `sample_ratio`: Ratio of entries to sample during eviction
- `min_ttl_seconds`: Minimum TTL for entries to be considered for eviction
- `max_eviction_ratio`: Maximum ratio of entries to evict in one round

### Metrics

Metrics collection settings:
- `enable`: Whether to enable metrics collection
- `level`: Metrics level (disabled, basic, detailed)
- `prometheus_port`: Port for Prometheus metrics endpoint
- `export_interval`: Interval for exporting metrics
- `histogram_buckets`: Latency histogram buckets (ms)

### Log

Logging settings:
- `level`: Log level (debug, info, warn, error)
- `format`: Log format (text, json)
- `output`: Log output (stdout, stderr, file)
- `file_path`: Log file path (when output is "file")
- `max_size_mb`: Maximum log file size before rotation
- `max_backups`: Maximum number of old log files to retain
- `max_age_days`: Maximum number of days to retain old log files

### Extensions

Extension-specific settings:
- `hot_reload`: Settings for hot reloading of configuration

## Usage

### Loading Configuration

```go
// Load from file
config, err := configs.LoadFromFile("configs/config.yaml")
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}

// Load with hot reloading using Viper
viperConfig, err := configs.LoadViperConfig("configs/config.yaml", true)
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}

// Subscribe to configuration changes
viperConfig.Subscribe(func(config *configs.Config) {
    log.Println("Configuration changed")
    // Update cache settings
})
```

### Creating Default Configuration

```go
// Create default configuration
config := configs.DefaultConfig()

// Customize configuration
config.Cache.MaxEntries = 1000000
config.Cache.DefaultTTL = 5 * time.Minute

// Save to file
if err := config.SaveToFile("configs/custom.yaml"); err != nil {
    log.Fatalf("Failed to save configuration: %v", err)
}
```

## Best Practices

1. Always validate configuration before using it
2. Provide sensible defaults for all settings
3. Use hot reloading for production environments
4. Document all configuration options
5. Use environment variables for sensitive information 