// Package loader defines interfaces for loading data into the cache
// when a cache miss occurs.
package loader

import (
	"context"
	"time"
)

// Loader is the interface for loading data into the cache when a key is not found.
// It provides a way to implement the cache-aside pattern.
type Loader interface {
	// Load retrieves data for the given key from a data source.
	// It returns the loaded value, a TTL for the cache entry, and any error encountered.
	// If the returned TTL is zero, the cache's default TTL will be used.
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - key: The key to load data for
	//
	// Returns:
	//   - interface{}: The loaded value
	//   - time.Duration: TTL for the cache entry (0 for default)
	//   - error: Error if the loading operation failed
	Load(ctx context.Context, key string) (interface{}, time.Duration, error)
}

// BatchLoader is the interface for loading multiple keys at once.
// This can be more efficient than loading keys individually when the data source
// supports batch operations.
type BatchLoader interface {
	// LoadBatch retrieves data for multiple keys from a data source.
	// It returns a map of keys to values, a map of keys to TTLs, and any error encountered.
	// If a TTL for a key is zero, the cache's default TTL will be used.
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - keys: The keys to load data for
	//
	// Returns:
	//   - map[string]interface{}: Map of keys to loaded values
	//   - map[string]time.Duration: Map of keys to TTLs (0 for default)
	//   - error: Error if the loading operation failed
	LoadBatch(ctx context.Context, keys []string) (map[string]interface{}, map[string]time.Duration, error)
}

// FunctionLoader wraps a function as a Loader.
// This is a convenient way to create a Loader from a function.
type FunctionLoader struct {
	loadFunc func(ctx context.Context, key string) (interface{}, time.Duration, error)
}

// Load calls the wrapped function.
func (l *FunctionLoader) Load(ctx context.Context, key string) (interface{}, time.Duration, error) {
	return l.loadFunc(ctx, key)
}

// NewFunctionLoader creates a new FunctionLoader from a function.
// The function should return the value and an error. The TTL will be set to the default.
//
// Parameters:
//   - fn: Function that loads data for a key
//
// Returns:
//   - *FunctionLoader: A new function-based loader
func NewFunctionLoader(fn func(ctx context.Context, key string) (interface{}, error)) *FunctionLoader {
	return &FunctionLoader{
		loadFunc: func(ctx context.Context, key string) (interface{}, time.Duration, error) {
			value, err := fn(ctx, key)
			return value, 0, err // Use default TTL
		},
	}
}

// NewFunctionLoaderWithTTL creates a new FunctionLoader from a function that specifies TTL.
//
// Parameters:
//   - fn: Function that loads data for a key and specifies TTL
//
// Returns:
//   - *FunctionLoader: A new function-based loader with TTL support
func NewFunctionLoaderWithTTL(fn func(ctx context.Context, key string) (interface{}, time.Duration, error)) *FunctionLoader {
	return &FunctionLoader{
		loadFunc: fn,
	}
}
