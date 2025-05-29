package cache

import (
	"context"
	"sync"
	"time"
)

// MockCache provides a simple in-memory cache implementation for testing.
type MockCache struct {
	name       string
	data       map[string]mockEntry
	mu         sync.RWMutex
	stats      Stats
	maxEntries int
	policy     string
}

type mockEntry struct {
	value     interface{}
	expiresAt time.Time
	accessed  time.Time
	hits      int
}

// NewMockCache creates a new mock cache for testing.
func NewMockCache(name string, maxEntries int, policy string) *MockCache {
	return &MockCache{
		name:       name,
		data:       make(map[string]mockEntry),
		maxEntries: maxEntries,
		policy:     policy,
	}
}

// Get retrieves a value from the cache.
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
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false, nil
	}

	// Update stats
	c.mu.Lock()
	entry.hits++
	entry.accessed = time.Now()
	c.data[key] = entry
	c.stats.Hits++
	c.mu.Unlock()

	return entry.value, true, nil
}

// Set adds a value to the cache with the specified TTL.
func (c *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict
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
func (c *MockCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]mockEntry)
	c.stats.EntryCount = 0
	return nil
}

// Stats returns statistics about the cache.
func (c *MockCache) Stats(ctx context.Context) (*Stats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Make a copy of the stats
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
func (c *MockCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = nil
	return nil
}

// evict removes an entry according to the eviction policy.
func (c *MockCache) evict() {
	if len(c.data) == 0 {
		return
	}

	var keyToEvict string

	switch c.policy {
	case "lru":
		// Find least recently accessed key
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
		leastHits := -1
		for k, entry := range c.data {
			if leastHits == -1 || entry.hits < leastHits {
				leastHits = entry.hits
				keyToEvict = k
			}
		}
	case "fifo":
		// Just take the first key we find (map iteration is random in Go)
		for k := range c.data {
			keyToEvict = k
			break
		}
	default: // random
		// Just take the first key we find (map iteration is random in Go)
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
