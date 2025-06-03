// Package loader provides interfaces for loading data into the cache
// when a cache miss occurs, supporting various back-source strategies.
//
// Package loader 提供接口用于在缓存未命中时将数据加载到缓存中，
// 支持各种回源策略。
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
//
// Loader 是包装基本Load方法的接口。
//
// Load 从数据源检索给定键的数据。
// 它返回加载的值、缓存条目的TTL以及遇到的任何错误。
// 如果返回的TTL为零，将使用缓存的默认TTL。
type Loader[T any] interface {
	Load(ctx context.Context, key string) (value T, ttl time.Duration, err error)
}

// LoaderFunc is a function type that implements the Loader interface.
//
// LoaderFunc 是实现Loader接口的函数类型。
type LoaderFunc[T any] func(ctx context.Context, key string) (T, time.Duration, error)

// Load calls the function itself.
//
// Load 调用函数本身。
func (f LoaderFunc[T]) Load(ctx context.Context, key string) (T, time.Duration, error) {
	return f(ctx, key)
}

// NewFunctionLoader creates a new Loader from a function that retrieves data.
// The function should return the value and an error. The TTL will be set to the default.
//
// NewFunctionLoader 从检索数据的函数创建一个新的Loader。
// 该函数应返回值和错误。TTL将设置为默认值。
func NewFunctionLoader[T any](fn func(ctx context.Context, key string) (T, error)) Loader[T] {
	return LoaderFunc[T](func(ctx context.Context, key string) (T, time.Duration, error) {
		value, err := fn(ctx, key)
		return value, 0, err // Use default TTL
	})
}

// NewFunctionLoaderWithTTL creates a new Loader from a function that retrieves data and specifies TTL.
//
// NewFunctionLoaderWithTTL 从检索数据并指定TTL的函数创建一个新的Loader。
func NewFunctionLoaderWithTTL[T any](fn func(ctx context.Context, key string) (T, time.Duration, error)) Loader[T] {
	return LoaderFunc[T](fn)
}

// BatchLoader is the interface for loading multiple keys at once.
//
// LoadBatch retrieves data for multiple keys from a data source.
// It returns a map of keys to values, a map of keys to TTLs, and any error encountered.
// If a TTL for a key is zero, the cache's default TTL will be used.
//
// BatchLoader 是用于一次加载多个键的接口。
//
// LoadBatch 从数据源检索多个键的数据。
// 它返回键到值的映射、键到TTL的映射以及遇到的任何错误。
// 如果键的TTL为零，将使用缓存的默认TTL。
type BatchLoader[T any] interface {
	LoadBatch(ctx context.Context, keys []string) (values map[string]T, ttls map[string]time.Duration, err error)
}

// BatchLoaderFunc is a function type that implements the BatchLoader interface.
//
// BatchLoaderFunc 是实现BatchLoader接口的函数类型。
type BatchLoaderFunc[T any] func(ctx context.Context, keys []string) (map[string]T, map[string]time.Duration, error)

// LoadBatch calls the function itself.
//
// LoadBatch 调用函数本身。
func (f BatchLoaderFunc[T]) LoadBatch(ctx context.Context, keys []string) (map[string]T, map[string]time.Duration, error) {
	return f(ctx, keys)
}

// FallbackLoader provides a fallback mechanism when the primary loader fails.
//
// FallbackLoader 提供当主加载器失败时的后备机制。
type FallbackLoader[T any] struct {
	Primary   Loader[T]
	Secondary Loader[T]
}

// Load attempts to load data using the primary loader.
// If the primary loader fails, it falls back to the secondary loader.
//
// Load 尝试使用主加载器加载数据。
// 如果主加载器失败，它会回退到次要加载器。
func (f *FallbackLoader[T]) Load(ctx context.Context, key string) (T, time.Duration, error) {
	value, ttl, err := f.Primary.Load(ctx, key)
	if err != nil && f.Secondary != nil {
		return f.Secondary.Load(ctx, key)
	}
	return value, ttl, err
}

// NewFallbackLoader creates a new FallbackLoader with the given primary and secondary loaders.
//
// NewFallbackLoader 使用给定的主加载器和次要加载器创建一个新的FallbackLoader。
func NewFallbackLoader[T any](primary, secondary Loader[T]) *FallbackLoader[T] {
	return &FallbackLoader[T]{
		Primary:   primary,
		Secondary: secondary,
	}
}

// CachedLoader wraps a loader with a local cache to reduce load on the backend.
//
// CachedLoader 用本地缓存包装加载器，以减轻后端负载。
type CachedLoader[T any] struct {
	Backend Loader[T]
	Cache   map[string]cachedItem[T]
	TTL     time.Duration
	mu      sync.RWMutex
}

// cachedItem represents an item in the local cache with its expiration time.
//
// cachedItem 表示本地缓存中的项目及其过期时间。
type cachedItem[T any] struct {
	Value      T
	Expiration time.Time
}

// Load attempts to retrieve the value from the local cache first.
// If the value is not in the cache or has expired, it loads from the backend.
//
// Load 首先尝试从本地缓存检索值。
// 如果值不在缓存中或已过期，它会从后端加载。
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
//
// NewCachedLoader 使用给定的后端加载器和TTL创建一个新的CachedLoader。
func NewCachedLoader[T any](backend Loader[T], ttl time.Duration) *CachedLoader[T] {
	return &CachedLoader[T]{
		Backend: backend,
		Cache:   make(map[string]cachedItem[T]),
		TTL:     ttl,
	}
}
