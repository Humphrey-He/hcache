// Package errors provides standardized error types for the cache.
// It defines common error types, error wrapping, and helper functions
// for error checking and handling in the cache implementation.
//
// Package errors 提供缓存的标准化错误类型。
// 它定义了常见错误类型、错误包装和用于缓存实现中错误检查和处理的辅助函数。
package errors

import (
	"errors"
	"fmt"
)

// Standard errors that can be returned by the cache.
// These provide consistent error types across the cache implementation.
//
// 缓存可能返回的标准错误。
// 这些提供了缓存实现中一致的错误类型。
var (
	// ErrNotFound is returned when a key is not found in the cache.
	// 当在缓存中找不到键时返回ErrNotFound。
	ErrNotFound = errors.New("cache: key not found")

	// ErrKeyEmpty is returned when an empty key is provided.
	// 当提供空键时返回ErrKeyEmpty。
	ErrKeyEmpty = errors.New("cache: key is empty")

	// ErrKeyTooLarge is returned when a key exceeds the maximum allowed size.
	// 当键超过允许的最大大小时返回ErrKeyTooLarge。
	ErrKeyTooLarge = errors.New("cache: key too large")

	// ErrValueTooLarge is returned when a value exceeds the maximum allowed size.
	// 当值超过允许的最大大小时返回ErrValueTooLarge。
	ErrValueTooLarge = errors.New("cache: value too large")

	// ErrCacheFull is returned when the cache is full and no items can be evicted.
	// 当缓存已满且无法淘汰任何项目时返回ErrCacheFull。
	ErrCacheFull = errors.New("cache: cache is full")

	// ErrInvalidTTL is returned when an invalid TTL is provided.
	// 当提供无效的TTL时返回ErrInvalidTTL。
	ErrInvalidTTL = errors.New("cache: invalid TTL")

	// ErrSerializationFailed is returned when value serialization fails.
	// 当值序列化失败时返回ErrSerializationFailed。
	ErrSerializationFailed = errors.New("cache: serialization failed")

	// ErrDeserializationFailed is returned when value deserialization fails.
	// 当值反序列化失败时返回ErrDeserializationFailed。
	ErrDeserializationFailed = errors.New("cache: deserialization failed")

	// ErrClosed is returned when an operation is performed on a closed cache.
	// 当对已关闭的缓存执行操作时返回ErrClosed。
	ErrClosed = errors.New("cache: cache is closed")

	// ErrAdmissionDenied is returned when a value is denied by the admission policy.
	// 当值被准入策略拒绝时返回ErrAdmissionDenied。
	ErrAdmissionDenied = errors.New("cache: admission denied")
)

// KeyError represents an error related to a specific key.
// It wraps an underlying error with the key that caused the error.
//
// KeyError 表示与特定键相关的错误。
// 它用导致错误的键包装底层错误。
type KeyError struct {
	Key string // The key that caused the error / 导致错误的键
	Err error  // The underlying error / 底层错误
}

// Error returns the error message.
// It implements the error interface.
//
// Error 返回错误消息。
// 它实现了error接口。
//
// Returns:
//   - string: The formatted error message including the key
func (e *KeyError) Error() string {
	return fmt.Sprintf("%s: %s", e.Err, e.Key)
}

// Unwrap returns the underlying error.
// This allows errors.Is and errors.As to work with wrapped errors.
//
// Unwrap 返回底层错误。
// 这允许errors.Is和errors.As与包装的错误一起工作。
//
// Returns:
//   - error: The underlying error
func (e *KeyError) Unwrap() error {
	return e.Err
}

// NewKeyError creates a new KeyError.
// It associates a key with an error.
//
// NewKeyError 创建一个新的KeyError。
// 它将键与错误关联起来。
//
// Parameters:
//   - key: The key that caused the error
//   - err: The underlying error
//
// Returns:
//   - *KeyError: A new key error instance
func NewKeyError(key string, err error) *KeyError {
	return &KeyError{Key: key, Err: err}
}

// IsNotFound returns true if the error indicates that a key was not found.
//
// IsNotFound 如果错误表示未找到键，则返回true。
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: True if the error is or wraps ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsKeyTooLarge returns true if the error indicates that a key is too large.
//
// IsKeyTooLarge 如果错误表示键太大，则返回true。
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: True if the error is or wraps ErrKeyTooLarge
func IsKeyTooLarge(err error) bool {
	return errors.Is(err, ErrKeyTooLarge)
}

// IsValueTooLarge returns true if the error indicates that a value is too large.
//
// IsValueTooLarge 如果错误表示值太大，则返回true。
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: True if the error is or wraps ErrValueTooLarge
func IsValueTooLarge(err error) bool {
	return errors.Is(err, ErrValueTooLarge)
}

// IsCacheFull returns true if the error indicates that the cache is full.
//
// IsCacheFull 如果错误表示缓存已满，则返回true。
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: True if the error is or wraps ErrCacheFull
func IsCacheFull(err error) bool {
	return errors.Is(err, ErrCacheFull)
}

// IsClosed returns true if the error indicates that the cache is closed.
//
// IsClosed 如果错误表示缓存已关闭，则返回true。
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: True if the error is or wraps ErrClosed
func IsClosed(err error) bool {
	return errors.Is(err, ErrClosed)
}

// IsAdmissionDenied returns true if the error indicates that admission was denied.
//
// IsAdmissionDenied 如果错误表示准入被拒绝，则返回true。
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: True if the error is or wraps ErrAdmissionDenied
func IsAdmissionDenied(err error) bool {
	return errors.Is(err, ErrAdmissionDenied)
}

// IsSerializationError returns true if the error is related to serialization.
//
// IsSerializationError 如果错误与序列化相关，则返回true。
//
// Parameters:
//   - err: The error to check
//
// Returns:
//   - bool: True if the error is or wraps ErrSerializationFailed or ErrDeserializationFailed
func IsSerializationError(err error) bool {
	return errors.Is(err, ErrSerializationFailed) || errors.Is(err, ErrDeserializationFailed)
}
