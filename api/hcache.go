// Package api provides the main entry point for the HCache API.
// It re-exports the core interfaces and types from the sub-packages.
package api

import (
	"github.com/Humphrey-He/hcache/api/codec"
	"github.com/Humphrey-He/hcache/api/core"
	"github.com/Humphrey-He/hcache/api/http"
	"github.com/Humphrey-He/hcache/api/loader"
	"github.com/Humphrey-He/hcache/api/metrics"
)

// Cache is the main interface for interacting with a cache.
// It is re-exported from the core package.
type Cache = core.Cache

// Stats represents cache statistics.
// It is re-exported from the core package.
type Stats = core.Stats

// Option is a function type for configuring a cache instance.
// It is re-exported from the core package.
type Option = core.Option

// Config holds the configuration parameters for a cache instance.
// It is re-exported from the core package.
type Config = core.Config

// Factory defines the interface for creating cache instances.
// It is re-exported from the core package.
type Factory = core.Factory

// Loader is the interface for loading data into the cache when a key is not found.
// It is re-exported from the loader package.
type Loader = loader.Loader

// BatchLoader is the interface for loading multiple keys at once.
// It is re-exported from the loader package.
type BatchLoader = loader.BatchLoader

// Codec defines the interface for encoding and decoding cache values.
// It is re-exported from the codec package.
type Codec = codec.Codec

// Compressor defines the interface for compressing and decompressing data.
// It is re-exported from the codec package.
type Compressor = codec.Compressor

// CacheMiddleware is an interface for HTTP middleware that caches responses.
// It is re-exported from the http package.
type CacheMiddleware = http.CacheMiddleware

// Collector defines the interface for collecting cache metrics.
// It is re-exported from the metrics package.
type Collector = metrics.Collector

// Reporter defines the interface for reporting cache metrics.
// It is re-exported from the metrics package.
type Reporter = metrics.Reporter

// MetricsLevel defines the level of detail for metrics collection.
// It is re-exported from the metrics package.
type MetricsLevel = metrics.MetricsLevel

// Re-export constants from the metrics package.
const (
	MetricsDisabled = metrics.MetricsDisabled
	MetricsBasic    = metrics.MetricsBasic
	MetricsDetailed = metrics.MetricsDetailed
)

// Re-export functions from the core package.
var (
	// WithMaxEntryCount sets the maximum number of entries the cache can hold.
	WithMaxEntryCount = core.WithMaxEntryCount

	// WithMaxMemoryBytes sets the maximum memory usage in bytes.
	WithMaxMemoryBytes = core.WithMaxMemoryBytes

	// WithTTL sets the default time-to-live for cache entries.
	WithTTL = core.WithTTL

	// WithEvictionPolicy sets the policy for evicting entries when the cache is full.
	WithEvictionPolicy = core.WithEvictionPolicy

	// WithShards sets the number of segments to divide the cache into.
	WithShards = core.WithShards

	// WithMetricsEnabled enables or disables performance metrics collection.
	WithMetricsEnabled = core.WithMetricsEnabled

	// WithCleanupInterval sets how often the cache checks for and removes expired entries.
	WithCleanupInterval = core.WithCleanupInterval

	// DefaultConfig returns a Config with reasonable default values.
	DefaultConfig = core.DefaultConfig

	// ApplyOptions applies the given options to a config.
	ApplyOptions = core.ApplyOptions

	// ValidateConfig checks if a configuration is valid.
	ValidateConfig = core.ValidateConfig
)

// Re-export functions from the loader package.
var (
	// NewFunctionLoader creates a new FunctionLoader from a function.
	NewFunctionLoader = loader.NewFunctionLoader

	// NewFunctionLoaderWithTTL creates a new FunctionLoader from a function that specifies TTL.
	NewFunctionLoaderWithTTL = loader.NewFunctionLoaderWithTTL
)

// Re-export error checking functions from the core package.
var (
	// IsNotFound returns true if the error is ErrNotFound or wraps ErrNotFound.
	IsNotFound = core.IsNotFound

	// IsInvalidKey returns true if the error is ErrInvalidKey or wraps ErrInvalidKey.
	IsInvalidKey = core.IsInvalidKey

	// IsInvalidValue returns true if the error is ErrInvalidValue or wraps ErrInvalidValue.
	IsInvalidValue = core.IsInvalidValue

	// IsCacheFull returns true if the error is ErrCacheFull or wraps ErrCacheFull.
	IsCacheFull = core.IsCacheFull

	// IsCacheClosed returns true if the error is ErrCacheClosed or wraps ErrCacheClosed.
	IsCacheClosed = core.IsCacheClosed

	// IsSerializationFailed returns true if the error is ErrSerializationFailed or wraps ErrSerializationFailed.
	IsSerializationFailed = core.IsSerializationFailed

	// IsDeserializationFailed returns true if the error is ErrDeserializationFailed or wraps ErrDeserializationFailed.
	IsDeserializationFailed = core.IsDeserializationFailed

	// IsLoaderNotConfigured returns true if the error is ErrLoaderNotConfigured or wraps ErrLoaderNotConfigured.
	IsLoaderNotConfigured = core.IsLoaderNotConfigured

	// IsLoaderFailed returns true if the error is ErrLoaderFailed or wraps ErrLoaderFailed.
	IsLoaderFailed = core.IsLoaderFailed
)
