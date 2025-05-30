// Package loader provides interfaces for loading data into the cache
// when a cache miss occurs, supporting various back-source strategies.
package loader

import (
	"context"
	"sync"
	"time"
)

// Loader is the interface that wraps the basic Load method.
//
// Load retrieves data for the given key from a data source.
// It returns the loaded value, a TTL for the cache entry, and any error encountered.
// If the returned TTL is zero, the cache's default TTL will be used.
type Loader[T any] interface {
	Load(ctx context.Context, key string) (value T, ttl time.Duration, err error)
}

// LoaderFunc is a function type that implements the Loader interface.
type LoaderFunc[T any] func(ctx context.Context, key string) (T, time.Duration, error)

// Load calls the function itself.
func (f LoaderFunc[T]) Load(ctx context.Context, key string) (T, time.Duration, error) {
	return f(ctx, key)
}

// NewFunctionLoader creates a new Loader from a function that retrieves data.
// The function should return the value and an error. The TTL will be set to the default.
func NewFunctionLoader[T any](fn func(ctx context.Context, key string) (T, error)) Loader[T] {
	return LoaderFunc[T](func(ctx context.Context, key string) (T, time.Duration, error) {
		value, err := fn(ctx, key)
		return value, 0, err // Use default TTL
	})
}

// NewFunctionLoaderWithTTL creates a new Loader from a function that retrieves data and specifies TTL.
func NewFunctionLoaderWithTTL[T any](fn func(ctx context.Context, key string) (T, time.Duration, error)) Loader[T] {
	return LoaderFunc[T](fn)
}

// BatchLoader is the interface for loading multiple keys at once.
//
// LoadBatch retrieves data for multiple keys from a data source.
// It returns a map of keys to values, a map of keys to TTLs, and any error encountered.
// If a TTL for a key is zero, the cache's default TTL will be used.
type BatchLoader[T any] interface {
	LoadBatch(ctx context.Context, keys []string) (values map[string]T, ttls map[string]time.Duration, err error)
}

// BatchLoaderFunc is a function type that implements the BatchLoader interface.
type BatchLoaderFunc[T any] func(ctx context.Context, keys []string) (map[string]T, map[string]time.Duration, error)

// LoadBatch calls the function itself.
func (f BatchLoaderFunc[T]) LoadBatch(ctx context.Context, keys []string) (map[string]T, map[string]time.Duration, error) {
	return f(ctx, keys)
}

// FallbackLoader provides a fallback mechanism when the primary loader fails.
type FallbackLoader[T any] struct {
	Primary   Loader[T]
	Secondary Loader[T]
}

// Load attempts to load data using the primary loader.
// If the primary loader fails, it falls back to the secondary loader.
func (f *FallbackLoader[T]) Load(ctx context.Context, key string) (T, time.Duration, error) {
	value, ttl, err := f.Primary.Load(ctx, key)
	if err != nil && f.Secondary != nil {
		return f.Secondary.Load(ctx, key)
	}
	return value, ttl, err
}

// NewFallbackLoader creates a new FallbackLoader with the given primary and secondary loaders.
func NewFallbackLoader[T any](primary, secondary Loader[T]) *FallbackLoader[T] {
	return &FallbackLoader[T]{
		Primary:   primary,
		Secondary: secondary,
	}
}

// CachedLoader wraps a loader with a local cache to reduce load on the backend.
type CachedLoader[T any] struct {
	Backend Loader[T]
	Cache   map[string]cachedItem[T]
	TTL     time.Duration
	mu      sync.RWMutex
}

type cachedItem[T any] struct {
	Value      T
	Expiration time.Time
}

// Load attempts to retrieve the value from the local cache first.
// If the value is not in the cache or has expired, it loads from the backend.
func (c *CachedLoader[T]) Load(ctx context.Context, key string) (T, time.Duration, error) {
	// Try to get from local cache first
	c.mu.RLock()
	if item, ok := c.Cache[key]; ok && time.Now().Before(item.Expiration) {
		c.mu.RUnlock()
		return item.Value, c.TTL, nil
	}
	c.mu.RUnlock()

	// Load from backend
	value, ttl, err := c.Backend.Load(ctx, key)
	if err != nil {
		return value, ttl, err
	}

	// Cache the result
	c.mu.Lock()
	c.Cache[key] = cachedItem[T]{
		Value:      value,
		Expiration: time.Now().Add(c.TTL),
	}
	c.mu.Unlock()

	return value, ttl, nil
}

// NewCachedLoader creates a new CachedLoader with the given backend loader and TTL.
func NewCachedLoader[T any](backend Loader[T], ttl time.Duration) *CachedLoader[T] {
	return &CachedLoader[T]{
		Backend: backend,
		Cache:   make(map[string]cachedItem[T]),
		TTL:     ttl,
	}
}
