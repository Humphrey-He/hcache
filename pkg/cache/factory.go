package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// basicCache is a simple in-memory cache implementation
type basicCache struct {
	name       string
	items      map[string]cacheItem
	mu         sync.RWMutex
	config     *Config
	stats      Stats
	statsLock  sync.RWMutex
	defaultTTL time.Duration
	dataLoader interface{} // This would be a proper loader type in full implementation
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewWithOptions creates a new cache instance with the provided options.
// It allows functional configuration of the cache.
//
// Parameters:
//   - name: The name of the cache instance
//   - options: A list of option functions to configure the cache
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the cache creation fails
func NewWithOptions(name string, options ...Option) (ICache, error) {
	config := NewDefaultConfig()
	config.Name = name

	// Apply all options
	for _, option := range options {
		option(config)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cache configuration: %w", err)
	}

	cache := &basicCache{
		name:       name,
		items:      make(map[string]cacheItem),
		config:     config,
		defaultTTL: config.TTL,
		dataLoader: config.Loader,
	}

	return cache, nil
}

// Implementation of ICache interface methods for basicCache

func (c *basicCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		c.recordMiss()
		return nil, false, nil
	}

	// Check if the item has expired
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		c.recordMiss()
		return nil, false, nil
	}

	c.recordHit()
	return item.value, true, nil
}

func (c *basicCache) GetOrLoad(ctx context.Context, key string) (interface{}, error) {
	// First try to get from cache
	value, found, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if found {
		return value, nil
	}

	// No loader configured, return error
	if c.dataLoader == nil {
		return nil, fmt.Errorf("key not found and no loader configured")
	}

	// In a real implementation, this would use the configured loader
	return nil, fmt.Errorf("loading not implemented for key: %s", key)
}

func (c *basicCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use the provided TTL, or the default if not specified
	expiration := time.Time{}
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	} else if ttl == 0 && c.defaultTTL > 0 {
		expiration = time.Now().Add(c.defaultTTL)
	}

	c.items[key] = cacheItem{
		value:      value,
		expiration: expiration,
	}

	return nil
}

func (c *basicCache) Delete(ctx context.Context, key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.items[key]
	if !exists {
		return false, nil
	}

	delete(c.items, key)
	return true, nil
}

func (c *basicCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
	return nil
}

func (c *basicCache) Stats(ctx context.Context) (*Stats, error) {
	c.statsLock.RLock()
	defer c.statsLock.RUnlock()

	// Create a copy of the stats to avoid concurrent modification
	statsCopy := Stats{
		EntryCount: int64(len(c.items)),
		Hits:       c.stats.Hits,
		Misses:     c.stats.Misses,
		Evictions:  c.stats.Evictions,
		Size:       c.stats.Size,
	}

	return &statsCopy, nil
}

func (c *basicCache) Close() error {
	// Clean up resources
	c.Clear(context.Background())
	return nil
}

func (c *basicCache) recordHit() {
	c.statsLock.Lock()
	defer c.statsLock.Unlock()
	c.stats.Hits++
}

func (c *basicCache) recordMiss() {
	c.statsLock.Lock()
	defer c.statsLock.Unlock()
	c.stats.Misses++
}

// New creates a new cache instance with the provided configuration.
// If config is nil, default configuration will be used.
//
// New 创建一个具有提供的配置的新缓存实例。
// 如果config为nil，将使用默认配置。
//
// Parameters:
//   - config: The configuration to use for the cache
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the cache creation fails
func New(config *Config) (ICache, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cache configuration: %w", err)
	}

	// The actual implementation will be provided by an internal package
	// This is just a placeholder for the public API
	//
	// 实际实现将由内部包提供
	// 这只是公共API的占位符
	return nil, fmt.Errorf("not implemented yet")
}

// NewFromJSON creates a new cache from a JSON configuration file.
// The JSON document must represent a valid cache configuration.
//
// NewFromJSON 从JSON配置文件创建新的缓存。
// JSON文档必须表示有效的缓存配置。
//
// Parameters:
//   - reader: An io.Reader providing the JSON configuration
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the configuration is invalid or the cache creation fails
func NewFromJSON(reader io.Reader) (ICache, error) {
	config := NewDefaultConfig()
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode JSON configuration: %w", err)
	}

	return New(config)
}

// NewFromYAML creates a new cache from a YAML configuration file.
// The YAML document must represent a valid cache configuration.
//
// NewFromYAML 从YAML配置文件创建新的缓存。
// YAML文档必须表示有效的缓存配置。
//
// Parameters:
//   - reader: An io.Reader providing the YAML configuration
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the configuration is invalid or the cache creation fails
func NewFromYAML(reader io.Reader) (ICache, error) {
	config := NewDefaultConfig()
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML configuration: %w", err)
	}

	return New(config)
}

// NewFromFile creates a new cache from a configuration file (JSON or YAML).
// The file format is determined by the file extension (.json, .yaml, or .yml).
//
// NewFromFile 从配置文件（JSON或YAML）创建新的缓存。
// 文件格式由文件扩展名确定（.json、.yaml或.yml）。
//
// Parameters:
//   - filename: The path to the configuration file
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the file cannot be read, the format is unsupported,
//     the configuration is invalid, or the cache creation fails
func NewFromFile(filename string) (ICache, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	// Determine file type based on extension
	// 根据扩展名确定文件类型
	switch {
	case hasExtension(filename, ".json"):
		return NewFromJSON(file)
	case hasExtension(filename, ".yaml"), hasExtension(filename, ".yml"):
		return NewFromYAML(file)
	default:
		return nil, fmt.Errorf("unsupported configuration file format: %s", filename)
	}
}

// hasExtension checks if a filename has the specified extension.
// The comparison is case-sensitive.
//
// hasExtension 检查文件名是否具有指定的扩展名。
// 比较区分大小写。
//
// Parameters:
//   - filename: The filename to check
//   - ext: The extension to check for (including the dot)
//
// Returns:
//   - bool: True if the filename has the specified extension
func hasExtension(filename, ext string) bool {
	if len(filename) < len(ext) {
		return false
	}
	return filename[len(filename)-len(ext):] == ext
}
