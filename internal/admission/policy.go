// Package admission provides cache admission control mechanisms.
// Package admission 提供缓存准入控制机制。
//
// Admission policies determine whether an item should be admitted into the cache
// based on its access patterns. This helps improve cache efficiency by only storing
// items that are likely to be accessed again, reducing cache pollution from one-time
// or infrequently accessed items.
//
// 准入策略根据访问模式确定是否应将项目添加到缓存中。通过仅存储可能再次被访问的项目，
// 这有助于提高缓存效率，减少一次性或不常访问项目对缓存的污染。
package admission

// Policy defines the interface for cache admission policies.
// Admission policies determine if a key-value pair should be added to the cache.
//
// Policy 定义缓存准入策略接口。
// 准入策略决定一个键值对是否应该被添加到缓存中。
type Policy interface {
	// Allow determines if a key should be admitted to the cache.
	// Returns true if admission is allowed, false if rejected.
	//
	// Allow 判断一个键是否应该被添加到缓存中。
	// 返回true表示允许添加，false表示拒绝。
	//
	// Parameters:
	//   - key: The key to evaluate for admission
	//
	// Returns:
	//   - bool: True if the key should be admitted, false otherwise
	Allow(key uint64) bool

	// Record registers an access to a key, updating internal statistics.
	// This is used to track access frequency for admission decisions.
	//
	// Record 记录一个键的访问，用于更新内部统计。
	// 这用于跟踪访问频率以做出准入决策。
	//
	// Parameters:
	//   - key: The key that was accessed
	Record(key uint64)

	// Reset clears the internal state of the admission policy.
	//
	// Reset 重置准入策略的内部状态。
	Reset()
}

// TinyLFUAdmission implements an admission policy based on the TinyLFU algorithm.
// It uses a Count-Min Sketch to estimate frequency and only admits items with
// frequency above a threshold.
//
// TinyLFUAdmission 实现基于TinyLFU的准入策略。
// 使用Count-Min Sketch估计频率，只有频率高于阈值的项目才会被缓存。
type TinyLFUAdmission struct {
	sketch    *CountMinSketch // Frequency estimator / 频率估计器
	threshold uint64          // Admission threshold / 准入阈值
}

// NewTinyLFUAdmission creates a new TinyLFU admission policy.
//
// NewTinyLFUAdmission 创建一个新的TinyLFU准入策略。
//
// Parameters:
//   - config: Configuration for the Count-Min Sketch
//
// Returns:
//   - *TinyLFUAdmission: A new TinyLFU admission policy
func NewTinyLFUAdmission(config *Config) *TinyLFUAdmission {
	sketch := NewCountMinSketch(config)

	return &TinyLFUAdmission{
		sketch:    sketch,
		threshold: 1, // Default threshold is 1, meaning an item must have been seen at least once
	}
}

// Allow determines if a key should be admitted to the cache based on its frequency.
//
// Allow 根据键的频率判断是否应该将其添加到缓存中。
//
// Parameters:
//   - key: The key to evaluate
//
// Returns:
//   - bool: True if the key's estimated frequency is at or above the threshold
func (t *TinyLFUAdmission) Allow(key uint64) bool {
	// Estimate the frequency of the key
	// 估计键的频率
	freq := t.sketch.Estimate(key)
	// Allow caching if frequency is at or above threshold
	// 如果频率大于或等于阈值，则允许缓存
	return freq >= t.threshold
}

// Record registers an access to a key, incrementing its frequency counter.
//
// Record 记录一个键的访问，增加其频率计数器。
//
// Parameters:
//   - key: The key that was accessed
func (t *TinyLFUAdmission) Record(key uint64) {
	t.sketch.Increment(key)
}

// Reset clears the internal state of the admission policy.
//
// Reset 重置准入策略的内部状态。
func (t *TinyLFUAdmission) Reset() {
	t.sketch.Reset()
}

// SetThreshold sets the admission threshold.
// Only items with a frequency at or above this threshold will be admitted.
//
// SetThreshold 设置准入阈值。
// 只有频率大于或等于此阈值的项目才会被允许加入缓存。
//
// Parameters:
//   - threshold: The new threshold value (must be greater than 0)
func (t *TinyLFUAdmission) SetThreshold(threshold uint64) {
	if threshold > 0 {
		t.threshold = threshold
	}
}

// NoAdmission implements a pass-through admission policy that always allows all keys.
// This is useful for scenarios where admission control is not needed.
//
// NoAdmission 实现一个无过滤的准入策略，总是允许所有键被缓存。
// 适用于不需要准入控制的场景。
type NoAdmission struct{}

// NewNoAdmission creates a new no-filter admission policy.
//
// NewNoAdmission 创建一个新的无过滤准入策略。
//
// Returns:
//   - *NoAdmission: A new no-filter admission policy
func NewNoAdmission() *NoAdmission {
	return &NoAdmission{}
}

// Allow always returns true, allowing all keys to be cached.
//
// Allow 总是返回true，允许所有键被缓存。
//
// Parameters:
//   - key: The key to evaluate (ignored)
//
// Returns:
//   - bool: Always true
func (n *NoAdmission) Allow(key uint64) bool {
	return true
}

// Record is a no-op since this policy doesn't track access patterns.
//
// Record 无操作，因为此策略不跟踪访问模式。
//
// Parameters:
//   - key: The key that was accessed (ignored)
func (n *NoAdmission) Record(key uint64) {
	// No operation
	// 无操作
}

// Reset is a no-op since this policy doesn't maintain state.
//
// Reset 无操作，因为此策略不维护状态。
func (n *NoAdmission) Reset() {
	// No operation
	// 无操作
}
