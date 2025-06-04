// Package http provides interfaces for integrating HCache with HTTP servers.
package http

import (
	"net/http"
	"time"

	"github.com/Humphrey-He/hcache/api/core"
)

// CacheMiddleware is an interface for HTTP middleware that caches responses.
type CacheMiddleware interface {
	// Handler wraps an HTTP handler with caching functionality.
	// It caches responses based on the request and returns cached responses when available.
	//
	// Parameters:
	//   - next: The next handler in the chain
	//
	// Returns:
	//   - http.Handler: A handler with caching functionality
	Handler(next http.Handler) http.Handler
}

// MiddlewareConfig holds configuration for the cache middleware.
type MiddlewareConfig struct {
	// Cache is the cache instance to use
	Cache core.Cache

	// TTL is the time-to-live for cached responses
	TTL time.Duration

	// KeyGenerator generates cache keys from HTTP requests
	KeyGenerator KeyGenerator

	// CachePredicate determines whether a request should be cached
	CachePredicate CachePredicate

	// VaryHeaders is a list of headers to include in the cache key
	VaryHeaders []string

	// StaleIfError determines whether to serve stale content on error
	StaleIfError bool

	// StaleWhileRevalidate enables background revalidation of stale entries
	StaleWhileRevalidate bool
}

// KeyGenerator generates cache keys from HTTP requests.
type KeyGenerator interface {
	// GenerateKey creates a cache key from an HTTP request.
	//
	// Parameters:
	//   - r: The HTTP request
	//
	// Returns:
	//   - string: The cache key
	GenerateKey(r *http.Request) string
}

// CachePredicate determines whether a request should be cached.
type CachePredicate interface {
	// ShouldCache determines whether a request should be cached.
	//
	// Parameters:
	//   - r: The HTTP request
	//
	// Returns:
	//   - bool: True if the request should be cached
	ShouldCache(r *http.Request) bool
}
