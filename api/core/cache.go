// Package core defines the core interfaces for the HCache API.
// It provides the fundamental contracts that all cache implementations must satisfy.
package core

import (
	"context"
	"time"
)

// Cache defines the primary interface for interacting with a cache.
// It provides methods for storing, retrieving, and managing cached data.
// All methods are designed to be thread-safe and support context-based cancellation.
type Cache interface {
	// Get retrieves a value from the cache.
	// It returns the value and a boolean indicating whether the key was found.
	// If the key is not found or has expired, (nil, false, nil) is returned.
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - key: The key to retrieve
	//
	// Returns:
	//   - interface{}: The cached value if found
	//   - bool: True if the key was found and is valid
	//   - error: Error if the retrieval operation failed
	Get(ctx context.Context, key string) (interface{}, bool, error)

	// Set adds a value to the cache with the specified TTL.
	// If the key already exists, its value is updated.
	// If ttl is 0, the default TTL from the configuration is used.
	// If ttl is negative, the entry does not expire.
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - key: The key under which to store the value
	//   - value: The value to store
	//   - ttl: Time-to-live for the entry
	//
	// Returns:
	//   - error: Error if the set operation failed
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache.
	// Returns true if the key was found and removed, false if the key was not found.
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - key: The key to remove
	//
	// Returns:
	//   - bool: True if the key was found and removed
	//   - error: Error if the delete operation failed
	Delete(ctx context.Context, key string) (bool, error)

	// Clear removes all values from the cache.
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//
	// Returns:
	//   - error: Error if the clear operation failed
	Clear(ctx context.Context) error

	// GetOrLoad retrieves a value from the cache, or loads it using the configured loader if not found.
	// This method provides a convenient way to implement the cache-aside pattern.
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//   - key: The key to retrieve
	//
	// Returns:
	//   - interface{}: The cached or loaded value
	//   - error: Error if the retrieval or loading operation failed
	GetOrLoad(ctx context.Context, key string) (interface{}, error)

	// Stats returns statistics about the cache.
	//
	// Parameters:
	//   - ctx: Context for the operation, can be used for cancellation
	//
	// Returns:
	//   - Stats: Cache statistics
	//   - error: Error if retrieving statistics failed
	Stats(ctx context.Context) (Stats, error)

	// Close releases resources used by the cache.
	// After calling Close, the cache should not be used anymore.
	//
	// Returns:
	//   - error: Error if the close operation failed
	Close() error
}

// Stats represents cache statistics and metrics.
type Stats struct {
	// EntryCount is the current number of entries in the cache
	EntryCount int64

	// Hits is the number of successful cache retrievals
	Hits int64

	// Misses is the number of cache retrievals where the key was not found
	Misses int64

	// Evictions is the number of entries removed due to capacity constraints
	Evictions int64

	// Size is the current memory usage of the cache in bytes
	Size int64
}
