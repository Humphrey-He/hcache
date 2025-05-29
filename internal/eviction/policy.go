// Package eviction provides cache eviction policy implementations.
// Package eviction 提供缓存淘汰策略实现。
//
// This package implements various eviction algorithms including LRU, LFU, and W-TinyLFU.
// These policies determine which items should be removed from the cache when it reaches capacity.
// The W-TinyLFU algorithm combines the benefits of both LFU and LRU to achieve better hit rates.
//
// 本包实现了多种淘汰算法，包括LRU、LFU和W-TinyLFU。
// 这些策略决定了当缓存达到容量上限时应该移除哪些条目。
// W-TinyLFU算法结合了LFU和LRU的优点，以实现更高的命中率。
package eviction

import (
	"container/list"
	"sync"
	"time"
)

// Entry represents an item stored in the cache.
// Entry 表示缓存中存储的一个条目。
type Entry struct {
	Key        uint64      // Key identifier / 键标识符
	Value      interface{} // Cached value / 缓存的值
	Size       int64       // Size of entry in bytes / 条目大小（字节）
	AccessTime int64       // Last access time (Unix nano) / 最后访问时间（Unix纳秒）
	Frequency  uint32      // Access frequency counter / 访问频率计数器
	ExpireAt   int64       // Expiration time (Unix nano, 0 means never expire) / 过期时间（Unix纳秒，0表示永不过期）
}

// Policy defines the interface for cache eviction policies.
// Policy 定义缓存淘汰策略接口。
type Policy interface {
	// Add adds a new entry to the policy.
	// Returns true if eviction is needed to make room.
	//
	// Add 添加一个新的条目。
	// 如果需要淘汰其他条目来腾出空间，则返回true。
	//
	// Parameters:
	//   - entry: The entry to add to the policy
	//
	// Returns:
	//   - bool: True if eviction is needed, false otherwise
	Add(entry *Entry) bool

	// Get retrieves an entry and updates its access status.
	// Returns nil if the entry doesn't exist.
	//
	// Get 获取一个条目，并更新其访问状态。
	// 如果条目不存在，则返回nil。
	//
	// Parameters:
	//   - key: The key to retrieve
	//
	// Returns:
	//   - *Entry: The retrieved entry or nil if not found
	Get(key uint64) *Entry

	// Remove removes an entry from the policy.
	// Returns the removed entry, or nil if the entry doesn't exist.
	//
	// Remove 从策略中移除一个条目。
	// 返回被移除的条目，如果条目不存在则返回nil。
	//
	// Parameters:
	//   - key: The key to remove
	//
	// Returns:
	//   - *Entry: The removed entry or nil if not found
	Remove(key uint64) *Entry

	// Evict evicts one or more entries to free up the specified amount of space.
	// Returns the list of evicted entries.
	//
	// Evict 淘汰一个或多个条目以释放指定大小的空间。
	// 返回被淘汰的条目列表。
	//
	// Parameters:
	//   - size: The amount of space to free up in bytes
	//
	// Returns:
	//   - []*Entry: List of evicted entries
	Evict(size int64) []*Entry

	// Len returns the number of entries in the policy.
	//
	// Len 返回策略中的条目数量。
	//
	// Returns:
	//   - int: Number of entries
	Len() int

	// Size returns the total size of all entries in bytes.
	//
	// Size 返回所有条目的总大小（字节）。
	//
	// Returns:
	//   - int64: Total size in bytes
	Size() int64

	// Clear removes all entries from the policy.
	//
	// Clear 清空策略中的所有条目。
	Clear()

	// Close releases resources used by the policy.
	//
	// Close 释放策略使用的资源。
	Close()
}

// Config defines configuration options for eviction policies.
// Config 定义淘汰策略的配置选项。
type Config struct {
	// MaxSize is the maximum capacity of the cache in bytes.
	// MaxSize 是缓存的最大容量（字节）。
	MaxSize int64

	// MaxItems is the maximum number of items the cache can hold.
	// MaxItems 是缓存可以容纳的最大条目数。
	MaxItems int

	// WindowRatio is the size ratio of the window cache (0-1).
	// WindowRatio 是窗口缓存的大小比例（0-1之间）。
	WindowRatio float64

	// EnableJanitor determines whether background cleanup of expired items is enabled.
	// EnableJanitor 决定是否启用后台清理过期条目。
	EnableJanitor bool

	// JanitorInterval is the interval between janitor runs in seconds.
	// JanitorInterval 是清理器运行的间隔时间（秒）。
	JanitorInterval int64
}

// BasePolicy provides a base implementation for eviction policies.
// BasePolicy 为淘汰策略提供基础实现。
type BasePolicy struct {
	mu          sync.RWMutex // Mutex for thread safety / 用于线程安全的互斥锁
	maxSize     int64        // Maximum capacity in bytes / 最大容量（字节）
	maxItems    int          // Maximum number of items / 最大条目数
	currentSize int64        // Current size in bytes / 当前大小（字节）
	itemCount   int          // Current number of items / 当前条目数
	janitor     *janitor     // Background cleanup process / 后台清理进程
}

// NewBasePolicy creates a new base eviction policy.
// NewBasePolicy 创建一个基础淘汰策略。
//
// Parameters:
//   - config: Configuration options for the policy
//
// Returns:
//   - *BasePolicy: A new base policy instance
func NewBasePolicy(config *Config) *BasePolicy {
	bp := &BasePolicy{
		maxSize:  config.MaxSize,
		maxItems: config.MaxItems,
	}

	// Create janitor if enabled
	// 如果启用了后台清理，则创建janitor
	if config.EnableJanitor && config.JanitorInterval > 0 {
		bp.janitor = newJanitor(time.Duration(config.JanitorInterval) * time.Second)
	}

	return bp
}

// ShouldEvict determines if eviction is needed.
// ShouldEvict 判断是否需要淘汰条目。
//
// Parameters:
//   - entrySize: Size of the entry being added
//
// Returns:
//   - bool: True if eviction is needed, false otherwise
func (bp *BasePolicy) ShouldEvict(entrySize int64) bool {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	// Check if maximum capacity would be exceeded
	// 如果设置了最大容量，则检查是否超过
	if bp.maxSize > 0 && bp.currentSize+entrySize > bp.maxSize {
		return true
	}

	// Check if maximum item count would be exceeded
	// 如果设置了最大条目数，则检查是否超过
	if bp.maxItems > 0 && bp.itemCount >= bp.maxItems {
		return true
	}

	return false
}

// UpdateSize updates the cache size and item count.
// UpdateSize 更新缓存大小和条目数。
//
// Parameters:
//   - sizeChange: Change in size (can be negative)
//   - countChange: Change in item count (can be negative)
func (bp *BasePolicy) UpdateSize(sizeChange int64, countChange int) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.currentSize += sizeChange
	bp.itemCount += countChange
}

// Size returns the current total size of the cache in bytes.
// Size 返回缓存当前占用的总大小（字节）。
//
// Returns:
//   - int64: Current size in bytes
func (bp *BasePolicy) Size() int64 {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return bp.currentSize
}

// Len returns the current number of items in the cache.
// Len 返回缓存中的当前条目数量。
//
// Returns:
//   - int: Number of items
func (bp *BasePolicy) Len() int {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	return bp.itemCount
}

// Close releases resources used by the policy.
// Close 释放策略使用的资源。
func (bp *BasePolicy) Close() {
	if bp.janitor != nil {
		bp.janitor.stop()
	}
}

// janitor handles background cleanup of expired entries.
// janitor 用于后台清理过期条目。
type janitor struct {
	interval time.Duration // Cleanup interval / 清理间隔
	stop     func()        // Function to stop the janitor / 停止清理器的函数
}

// newJanitor creates a new janitor for background cleanup.
// newJanitor 创建一个新的后台清理器。
//
// Parameters:
//   - interval: Time between cleanup runs
//
// Returns:
//   - *janitor: A new janitor instance
func newJanitor(interval time.Duration) *janitor {
	stopChan := make(chan struct{})
	j := &janitor{
		interval: interval,
		stop: func() {
			close(stopChan)
		},
	}

	return j
}

// KeyValue represents a key-value pair for passing between eviction policies.
// KeyValue 表示在淘汰策略之间传递的键值对。
type KeyValue struct {
	Key   uint64      // Key identifier / 键标识符
	Value interface{} // Associated value / 关联的值
}

// Node represents a node in a linked list, used for LRU/LFU implementation.
// Node 表示链表中的一个节点，用于LRU/LFU实现。
type Node struct {
	Key        uint64        // Key identifier / 键标识符
	Value      interface{}   // Cached value / 缓存的值
	Size       int64         // Size in bytes / 大小（字节）
	AccessTime int64         // Last access time / 最后访问时间
	Frequency  uint32        // Access frequency / 访问频率
	ExpireAt   int64         // Expiration time / 过期时间
	listElem   *list.Element // Reference to list element / 链表元素的引用
}

// IsExpired checks if the node has expired.
// IsExpired 判断节点是否已过期。
//
// Returns:
//   - bool: True if expired, false otherwise
func (n *Node) IsExpired() bool {
	if n.ExpireAt == 0 {
		return false
	}
	return time.Now().UnixNano() > n.ExpireAt
}

// ToEntry converts a Node to an Entry.
// ToEntry 将节点转换为Entry。
//
// Returns:
//   - *Entry: Entry representation of the node
func (n *Node) ToEntry() *Entry {
	return &Entry{
		Key:        n.Key,
		Value:      n.Value,
		Size:       n.Size,
		AccessTime: n.AccessTime,
		Frequency:  n.Frequency,
		ExpireAt:   n.ExpireAt,
	}
}
