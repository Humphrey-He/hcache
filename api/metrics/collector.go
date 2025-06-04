// Package metrics provides interfaces for collecting and reporting
// cache performance metrics.
package metrics

import (
	"time"
)

// Collector defines the interface for collecting cache metrics.
// It provides methods for recording various cache events.
type Collector interface {
	// RecordHit records a cache hit.
	//
	// Parameters:
	//   - key: The key that was hit
	//   - latency: How long it took to retrieve the value
	RecordHit(key string, latency time.Duration)

	// RecordMiss records a cache miss.
	//
	// Parameters:
	//   - key: The key that was missed
	RecordMiss(key string)

	// RecordEviction records a cache eviction.
	//
	// Parameters:
	//   - key: The key that was evicted
	//   - reason: The reason for eviction (capacity, ttl, etc.)
	RecordEviction(key string, reason string)

	// RecordSet records a cache set operation.
	//
	// Parameters:
	//   - key: The key that was set
	//   - size: The size of the value in bytes
	//   - latency: How long it took to set the value
	RecordSet(key string, size int, latency time.Duration)

	// RecordDelete records a cache delete operation.
	//
	// Parameters:
	//   - key: The key that was deleted
	RecordDelete(key string)

	// RecordError records a cache operation error.
	//
	// Parameters:
	//   - operation: The operation that failed (get, set, etc.)
	//   - err: The error that occurred
	RecordError(operation string, err error)
}

// Reporter defines the interface for reporting cache metrics.
// It provides methods for retrieving collected metrics.
type Reporter interface {
	// GetHitRate returns the cache hit rate.
	//
	// Returns:
	//   - float64: The hit rate (0.0 to 1.0)
	GetHitRate() float64

	// GetHitCount returns the number of cache hits.
	//
	// Returns:
	//   - int64: The hit count
	GetHitCount() int64

	// GetMissCount returns the number of cache misses.
	//
	// Returns:
	//   - int64: The miss count
	GetMissCount() int64

	// GetEvictionCount returns the number of cache evictions.
	//
	// Returns:
	//   - int64: The eviction count
	GetEvictionCount() int64

	// GetAverageGetLatency returns the average latency of get operations.
	//
	// Returns:
	//   - time.Duration: The average latency
	GetAverageGetLatency() time.Duration

	// GetAverageSetLatency returns the average latency of set operations.
	//
	// Returns:
	//   - time.Duration: The average latency
	GetAverageSetLatency() time.Duration

	// GetSize returns the current size of the cache in bytes.
	//
	// Returns:
	//   - int64: The cache size in bytes
	GetSize() int64

	// GetEntryCount returns the current number of entries in the cache.
	//
	// Returns:
	//   - int64: The number of entries
	GetEntryCount() int64

	// Reset resets all metrics.
	Reset()
}

// MetricsLevel defines the level of detail for metrics collection.
type MetricsLevel string

const (
	// MetricsDisabled disables metrics collection.
	MetricsDisabled MetricsLevel = "disabled"

	// MetricsBasic enables basic metrics collection (hits, misses, etc.).
	MetricsBasic MetricsLevel = "basic"

	// MetricsDetailed enables detailed metrics collection (latency distributions, etc.).
	MetricsDetailed MetricsLevel = "detailed"
)
