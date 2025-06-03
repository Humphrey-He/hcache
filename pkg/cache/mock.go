package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockCache provides a simple in-memory cache implementation for testing.
//
// MockCache 提供一个简单的内存缓存实现，用于测试。
type MockCache struct {
	name       string
	data       map[string]mockEntry
	mu         sync.RWMutex
	stats      Stats
	maxEntries int
	policy     string
	dataLoader interface{} // This would be a proper loader type in full implementation
}

// mockEntry represents an item in the mock cache.
//
// mockEntry 表示模拟缓存中的一个项目。
type mockEntry struct {
	value     interface{}
	expiresAt time.Time
	accessed  time.Time
	hits      int
}

// NewMockCache creates a new mock cache for testing.
//
// NewMockCache 创建一个新的模拟缓存，用于测试。
//
// Parameters:
//   - name: The name of the cache
//   - maxEntries: The maximum number of entries the cache can hold
//   - policy: The eviction policy to use
//
// Returns:
//   - *MockCache: A new mock cache instance
func NewMockCache(name string, maxEntries int, policy string) *MockCache {
	return &MockCache{
		name:       name,
		data:       make(map[string]mockEntry),
		maxEntries: maxEntries,
		policy:     policy,
	}
}

// Get retrieves a value from the cache.
//
// Get 从缓存中检索值。
//
// Parameters:
//   - ctx: Context for the operation
//   - key: The key to retrieve
//
// Returns:
//   - interface{}: The value if found
//   - bool: True if the key was found and is valid
//   - error: Error if the retrieval operation failed
func (c *MockCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	c.mu.RLock()
	entry, exists := c.data[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false, nil
	}

	// Check if expired
	// 检查是否过期
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false, nil
	}

	// Update stats
	// 更新统计信息
	c.mu.Lock()
	entry.hits++
	entry.accessed = time.Now()
	c.data[key] = entry
	c.stats.Hits++
	c.mu.Unlock()

	return entry.value, true, nil
}

// Set adds a value to the cache with the specified TTL.
//
// Set 将值添加到缓存中，并指定TTL。
//
// Parameters:
//   - ctx: Context for the operation
//   - key: The key under which to store the value
//   - value: The value to store
//   - ttl: Time-to-live for the entry
//
// Returns:
//   - error: Error if the set operation failed
func (c *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict
	// 检查是否需要淘汰
	if len(c.data) >= c.maxEntries && c.maxEntries > 0 {
		c.evict()
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.data[key] = mockEntry{
		value:     value,
		expiresAt: expiresAt,
		accessed:  time.Now(),
		hits:      0,
	}

	c.stats.EntryCount = int64(len(c.data))
	return nil
}

// Delete removes a value from the cache.
//
// Delete 从缓存中删除值。
//
// Parameters:
//   - ctx: Context for the operation
//   - key: The key to remove
//
// Returns:
//   - bool: True if the key was found and removed
//   - error: Error if the delete operation failed
func (c *MockCache) Delete(ctx context.Context, key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.data[key]
	if exists {
		delete(c.data, key)
		c.stats.EntryCount = int64(len(c.data))
		return true, nil
	}
	return false, nil
}

// Clear removes all values from the cache.
//
// Clear 删除缓存中的所有值。
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - error: Error if the clear operation failed
func (c *MockCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]mockEntry)
	c.stats.EntryCount = 0
	return nil
}

// Stats returns statistics about the cache.
//
// Stats 返回有关缓存的统计信息。
//
// Parameters:
//   - ctx: Context for the operation
//
// Returns:
//   - *Stats: Cache statistics
//   - error: Error if retrieving statistics failed
func (c *MockCache) Stats(ctx context.Context) (*Stats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Make a copy of the stats
	// 复制统计信息
	statsCopy := Stats{
		EntryCount: c.stats.EntryCount,
		Hits:       c.stats.Hits,
		Misses:     c.stats.Misses,
		Evictions:  c.stats.Evictions,
		Size:       int64(len(c.data)),
	}

	return &statsCopy, nil
}

// Close cleans up resources used by the cache.
//
// Close 清理缓存使用的资源。
//
// Returns:
//   - error: Error if the close operation failed
func (c *MockCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = nil
	return nil
}

// evict removes an entry according to the eviction policy.
//
// evict 根据淘汰策略删除一个条目。
func (c *MockCache) evict() {
	if len(c.data) == 0 {
		return
	}

	var keyToEvict string

	switch c.policy {
	case "lru":
		// Find least recently accessed key
		// 查找最近最少访问的键
		var oldestAccess time.Time
		first := true
		for k, entry := range c.data {
			if first || entry.accessed.Before(oldestAccess) {
				oldestAccess = entry.accessed
				keyToEvict = k
				first = false
			}
		}
	case "lfu":
		// Find least frequently used key
		// 查找最不常用的键
		leastHits := -1
		for k, entry := range c.data {
			if leastHits == -1 || entry.hits < leastHits {
				leastHits = entry.hits
				keyToEvict = k
			}
		}
	case "fifo":
		// Just take the first key we find (map iteration is random in Go)
		// 只取我们找到的第一个键（Go中的映射迭代是随机的）
		for k := range c.data {
			keyToEvict = k
			break
		}
	default: // random
		// Just take the first key we find (map iteration is random in Go)
		// 只取我们找到的第一个键（Go中的映射迭代是随机的）
		for k := range c.data {
			keyToEvict = k
			break
		}
	}

	if keyToEvict != "" {
		delete(c.data, keyToEvict)
		c.stats.Evictions++
	}
}

// GetOrLoad retrieves a value from the cache, or returns an error if not found.
// This mock implementation doesn't actually load data from anywhere.
//
// GetOrLoad 从缓存中检索值，如果未找到则返回错误。
// 这个模拟实现实际上不会从任何地方加载数据。
//
// Parameters:
//   - ctx: Context for the operation
//   - key: The key to retrieve
//
// Returns:
//   - interface{}: The value if found
//   - error: Error if the key was not found or the operation failed
func (c *MockCache) GetOrLoad(ctx context.Context, key string) (interface{}, error) {
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
	return nil, fmt.Errorf("mock cache does not support loading for key: %s", key)
}
