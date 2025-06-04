package core

import "errors"

// Standard errors returned by the cache.
var (
	// ErrNotFound is returned when a key is not found in the cache.
	ErrNotFound = errors.New("key not found in cache")

	// ErrInvalidKey is returned when a key is invalid.
	ErrInvalidKey = errors.New("invalid key")

	// ErrInvalidValue is returned when a value cannot be stored in the cache.
	ErrInvalidValue = errors.New("invalid value")

	// ErrCacheFull is returned when the cache is full and cannot accept more entries.
	ErrCacheFull = errors.New("cache is full")

	// ErrCacheClosed is returned when an operation is attempted on a closed cache.
	ErrCacheClosed = errors.New("cache is closed")

	// ErrSerializationFailed is returned when value serialization fails.
	ErrSerializationFailed = errors.New("value serialization failed")

	// ErrDeserializationFailed is returned when value deserialization fails.
	ErrDeserializationFailed = errors.New("value deserialization failed")

	// ErrLoaderNotConfigured is returned when GetOrLoad is called but no loader is configured.
	ErrLoaderNotConfigured = errors.New("data loader not configured")

	// ErrLoaderFailed is returned when the data loader fails to load a value.
	ErrLoaderFailed = errors.New("data loader failed")
)

// IsNotFound returns true if the error is ErrNotFound or wraps ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsInvalidKey returns true if the error is ErrInvalidKey or wraps ErrInvalidKey.
func IsInvalidKey(err error) bool {
	return errors.Is(err, ErrInvalidKey)
}

// IsInvalidValue returns true if the error is ErrInvalidValue or wraps ErrInvalidValue.
func IsInvalidValue(err error) bool {
	return errors.Is(err, ErrInvalidValue)
}

// IsCacheFull returns true if the error is ErrCacheFull or wraps ErrCacheFull.
func IsCacheFull(err error) bool {
	return errors.Is(err, ErrCacheFull)
}

// IsCacheClosed returns true if the error is ErrCacheClosed or wraps ErrCacheClosed.
func IsCacheClosed(err error) bool {
	return errors.Is(err, ErrCacheClosed)
}

// IsSerializationFailed returns true if the error is ErrSerializationFailed or wraps ErrSerializationFailed.
func IsSerializationFailed(err error) bool {
	return errors.Is(err, ErrSerializationFailed)
}

// IsDeserializationFailed returns true if the error is ErrDeserializationFailed or wraps ErrDeserializationFailed.
func IsDeserializationFailed(err error) bool {
	return errors.Is(err, ErrDeserializationFailed)
}

// IsLoaderNotConfigured returns true if the error is ErrLoaderNotConfigured or wraps ErrLoaderNotConfigured.
func IsLoaderNotConfigured(err error) bool {
	return errors.Is(err, ErrLoaderNotConfigured)
}

// IsLoaderFailed returns true if the error is ErrLoaderFailed or wraps ErrLoaderFailed.
func IsLoaderFailed(err error) bool {
	return errors.Is(err, ErrLoaderFailed)
}
