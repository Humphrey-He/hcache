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
//
// basicCache 是一个简单的内存缓存实现
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

// cacheItem represents a single item in the cache with its value and expiration time
//
// cacheItem 表示缓存中的单个项目及其值和过期时间
type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewWithOptions creates a new cache instance with the provided options.
// It allows functional configuration of the cache.
//
// NewWithOptions 创建一个具有提供的选项的新缓存实例。
// 它允许缓存的函数式配置。
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
		defaultTTL: config.DefaultTTL,
		dataLoader: config.Loader,
	}

	return cache, nil
}

// Implementation of ICache interface methods for basicCache

// Get retrieves a value from the cache.
//
// Get 从缓存中检索值。
func (c *basicCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		c.recordMiss()
		return nil, false, nil
	}

	// Check if the item has expired
	// 检查项目是否已过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		c.recordMiss()
		return nil, false, nil
	}

	c.recordHit()
	return item.value, true, nil
}

// GetOrLoad retrieves a value from the cache or loads it if not found.
//
// GetOrLoad 从缓存中检索值，如果未找到则加载它。
func (c *basicCache) GetOrLoad(ctx context.Context, key string) (interface{}, error) {
	// First try to get from cache
	// 首先尝试从缓存获取
	value, found, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if found {
		return value, nil
	}

	// No loader configured, return error
	// 没有配置加载器，返回错误
	if c.dataLoader == nil {
		return nil, fmt.Errorf("key not found and no loader configured")
	}

	// In a real implementation, this would use the configured loader
	// 在实际实现中，这将使用配置的加载器
	return nil, fmt.Errorf("loading not implemented for key: %s", key)
}

// Set adds a value to the cache with the specified TTL.
//
// Set 将值添加到缓存中，并指定TTL。
func (c *basicCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use the provided TTL, or the default if not specified
	// 使用提供的TTL，如果未指定则使用默认值
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

// Delete removes a value from the cache.
//
// Delete 从缓存中删除值。
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

// Clear removes all values from the cache.
//
// Clear 删除缓存中的所有值。
func (c *basicCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]cacheItem)
	return nil
}

// Stats returns statistics about the cache.
//
// Stats 返回有关缓存的统计信息。
func (c *basicCache) Stats(ctx context.Context) (*Stats, error) {
	c.statsLock.RLock()
	defer c.statsLock.RUnlock()

	// Create a copy of the stats to avoid concurrent modification
	// 创建统计信息的副本以避免并发修改
	statsCopy := Stats{
		EntryCount: int64(len(c.items)),
		Hits:       c.stats.Hits,
		Misses:     c.stats.Misses,
		Evictions:  c.stats.Evictions,
		Size:       c.stats.Size,
	}

	return &statsCopy, nil
}

// Close cleans up resources used by the cache.
//
// Close 清理缓存使用的资源。
func (c *basicCache) Close() error {
	// Clean up resources
	// 清理资源
	c.Clear(context.Background())
	return nil
}

// recordHit increments the hit counter in the cache statistics.
//
// recordHit 增加缓存统计中的命中计数器。
func (c *basicCache) recordHit() {
	c.statsLock.Lock()
	defer c.statsLock.Unlock()
	c.stats.Hits++
}

// recordMiss increments the miss counter in the cache statistics.
//
// recordMiss 增加缓存统计中的未命中计数器。
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

	cache := &basicCache{
		name:       config.Name,
		items:      make(map[string]cacheItem),
		config:     config,
		defaultTTL: config.DefaultTTL,
		dataLoader: config.Loader,
	}

	return cache, nil
}

// NewFromJSON creates a new cache instance from a JSON configuration.
// The JSON data is read from the provided reader.
//
// NewFromJSON 从JSON配置创建新的缓存实例。
// JSON数据从提供的读取器中读取。
//
// Parameters:
//   - reader: An io.Reader providing the JSON configuration data
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the configuration parsing or cache creation fails
func NewFromJSON(reader io.Reader) (ICache, error) {
	var config Config
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode JSON configuration: %w", err)
	}

	return New(&config)
}

// NewFromYAML creates a new cache instance from a YAML configuration.
// The YAML data is read from the provided reader.
//
// NewFromYAML 从YAML配置创建新的缓存实例。
// YAML数据从提供的读取器中读取。
//
// Parameters:
//   - reader: An io.Reader providing the YAML configuration data
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the configuration parsing or cache creation fails
func NewFromYAML(reader io.Reader) (ICache, error) {
	var config Config
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML configuration: %w", err)
	}

	return New(&config)
}

// NewFromFile creates a new cache instance from a configuration file.
// The file format (JSON or YAML) is determined by the file extension.
//
// NewFromFile 从配置文件创建新的缓存实例。
// 文件格式（JSON或YAML）由文件扩展名确定。
//
// Parameters:
//   - filename: The path to the configuration file
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the file reading, parsing, or cache creation fails
func NewFromFile(filename string) (ICache, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	// Determine format from file extension
	// 从文件扩展名确定格式
	if hasExtension(filename, ".json") {
		return NewFromJSON(file)
	} else if hasExtension(filename, ".yaml") || hasExtension(filename, ".yml") {
		return NewFromYAML(file)
	}

	return nil, fmt.Errorf("unsupported file format for %s", filename)
}

// hasExtension checks if a filename has the specified extension.
//
// hasExtension 检查文件名是否具有指定的扩展名。
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
