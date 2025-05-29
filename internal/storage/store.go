// Package storage provides high-performance sharded storage implementation.
// Package storage 提供高性能的分片存储实现。
//
// This package serves as the core storage layer for the cache, supporting high-concurrency
// read/write operations and atomic operations. It uses sharding to reduce lock contention
// and improve performance in multi-threaded environments. The storage is optimized for
// fast lookups and efficient memory usage.
//
// 本包作为缓存的核心存储层，支持高并发读写和原子操作。它使用分片技术来减少锁竞争，
// 提高多线程环境中的性能。存储层针对快速查找和高效内存使用进行了优化。
package storage

import (
	"hash/fnv"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	// defaultShardCount is the default number of shards.
	// Power of 2 is chosen to optimize modulo operations.
	//
	// defaultShardCount 是默认分片数量，选择2的幂次方以优化取模运算。
	defaultShardCount = 256

	// defaultLoadFactor is the default load factor.
	// When the number of elements in a shard exceeds capacity*loadFactor, it triggers a resize.
	//
	// defaultLoadFactor 是默认负载因子，当单个分片中的元素数量超过容量*负载因子时触发扩容。
	defaultLoadFactor = 0.75

	// defaultInitialCapacity is the default initial capacity per shard.
	//
	// defaultInitialCapacity 是每个分片的默认初始容量。
	defaultInitialCapacity = 64

	// defaultBatchSize is the default buffer size for batch operations.
	//
	// defaultBatchSize 是批量写入的默认缓冲区大小。
	defaultBatchSize = 32
)

// Item represents an entry stored in the cache.
// Item 表示存储在缓存中的条目。
type Item struct {
	Key        uint64      // Unique identifier / 键
	Value      interface{} // Cached value / 值
	Size       int64       // Size in bytes / 条目大小（字节）
	AccessTime int64       // Last access timestamp (Unix nano) / 最后访问时间（Unix纳秒）
	ExpireAt   int64       // Expiration timestamp (Unix nano, 0 means never expire) / 过期时间（Unix纳秒，0表示永不过期）
	Cost       int64       // Cost value for admission control / 成本，用于容量控制
	Flags      uint32      // Bit flags for item status / 标志位，用于标记条目状态
}

// IsExpired checks if the item has expired.
//
// IsExpired 判断条目是否已过期。
//
// Returns:
//   - bool: True if the item has expired, false otherwise
func (item *Item) IsExpired() bool {
	if item.ExpireAt == 0 {
		return false
	}
	return time.Now().UnixNano() > item.ExpireAt
}

// Clone creates a copy of the item.
//
// Clone 创建条目的副本。
//
// Returns:
//   - *Item: A new item with the same values
func (item *Item) Clone() *Item {
	return &Item{
		Key:        item.Key,
		Value:      item.Value,
		Size:       item.Size,
		AccessTime: item.AccessTime,
		ExpireAt:   item.ExpireAt,
		Cost:       item.Cost,
		Flags:      item.Flags,
	}
}

// Config contains configuration options for the storage.
// Config 存储层配置选项。
type Config struct {
	// ShardCount is the number of shards, recommended to be a power of 2.
	// ShardCount 是分片数量，建议为2的幂次方。
	ShardCount int

	// InitialCapacity is the initial capacity per shard.
	// InitialCapacity 是每个分片的初始容量。
	InitialCapacity int

	// LoadFactor determines when to resize the internal maps.
	// LoadFactor 决定何时调整内部映射的大小。
	LoadFactor float64

	// TrackAccessTime enables tracking of access times for items.
	// TrackAccessTime 是否启用访问时间追踪。
	TrackAccessTime bool

	// BatchSize is the buffer size for batch operations.
	// BatchSize 是批量操作的缓冲区大小。
	BatchSize int

	// MaxMemoryBytes is the maximum memory usage in bytes.
	// MaxMemoryBytes 是最大内存使用量（字节）。
	MaxMemoryBytes int64

	// AsyncAccessUpdate enables asynchronous updates of access times.
	// AsyncAccessUpdate 是否启用异步更新访问时间。
	AsyncAccessUpdate bool
}

// Store is the main implementation of sharded storage.
// It reduces lock contention by dividing the keyspace into multiple shards.
//
// Store 是分片存储的主要实现。
// 通过将键空间分割为多个分片来减少锁竞争，提高并发性能。
type Store struct {
	shards     []*shard      // Array of shards / 分片数组
	shardCount uint64        // Number of shards / 分片数量
	shardMask  uint64        // Mask for fast shard index calculation / 分片掩码，用于快速计算分片索引
	config     *Config       // Storage configuration / 存储配置
	stats      *Stats        // Statistics collector / 统计信息
	totalSize  int64         // Total size in bytes / 总大小（字节）
	itemCount  int64         // Total item count / 总条目数
	asyncQueue chan asyncOp  // Queue for async operations / 异步操作队列
	closeChan  chan struct{} // Channel for shutdown signaling / 关闭信号
	closeOnce  sync.Once     // Ensures close happens only once / 确保只关闭一次
}

// Stats collects storage statistics.
// Stats 存储统计信息。
type Stats struct {
	Hits            uint64 // Cache hit count / 命中次数
	Misses          uint64 // Cache miss count / 未命中次数
	Evictions       uint64 // Eviction count / 淘汰次数
	Expirations     uint64 // Expiration count / 过期次数
	ItemCount       int64  // Current item count / 条目数量
	BytesSize       int64  // Current size in bytes / 总大小（字节）
	EvictionCost    int64  // Cost of evicted items / 淘汰成本
	EvictionCount   uint64 // Number of eviction operations / 淘汰次数
	ExpiredCount    uint64 // Number of expired items / 过期条目数
	ConflictCount   uint64 // Hash conflict count / 冲突次数
	OverwriteCount  uint64 // Overwrite count / 覆盖次数
	AsyncQueueSize  int64  // Size of async operation queue / 异步队列大小
	AsyncDropCount  uint64 // Count of dropped async operations / 异步丢弃次数
	ReadLatencyNs   int64  // Average read latency in ns / 读取延迟（纳秒）
	WriteLatencyNs  int64  // Average write latency in ns / 写入延迟（纳秒）
	DeleteLatencyNs int64  // Average delete latency in ns / 删除延迟（纳秒）
}

// shard represents a single partition of the storage.
// shard 表示存储的一个分片。
type shard struct {
	sync.RWMutex
	items        map[uint64]*Item // Map of items in this shard / 存储条目的映射
	initialSize  int              // Initial capacity / 初始大小
	loadFactor   float64          // Load factor for resizing / 负载因子
	trackAccess  bool             // Whether to track access times / 是否追踪访问时间
	evictionList *list            // List for eviction ordering / 淘汰列表
}

// asyncOp represents an asynchronous operation.
// asyncOp 表示异步操作。
type asyncOp struct {
	op     int        // Operation type / 操作类型
	key    uint64     // Key for the operation / 键
	value  *Item      // Value for set operations / 值
	result chan error // Channel for operation result / 结果通道
}

// asyncOpType defines types of asynchronous operations.
// asyncOpType 异步操作类型。
const (
	opAccessUpdate = iota // Update access time / 更新访问时间
	opSet                 // Set an item / 设置条目
	opDelete              // Delete an item / 删除条目
)

// list is a simple doubly-linked list for maintaining eviction order.
// list 简单的双向链表，用于维护淘汰顺序。
type list struct {
	head *listNode // First node / 头节点
	tail *listNode // Last node / 尾节点
	size int       // Number of nodes / 大小
}

// listNode is a node in the doubly-linked list.
// listNode 链表节点。
type listNode struct {
	key  uint64    // Key of the item / 键
	prev *listNode // Previous node / 前一个节点
	next *listNode // Next node / 后一个节点
}

// NewStore creates a new storage instance.
//
// NewStore 创建一个新的存储实例。
//
// Parameters:
//   - config: Configuration options for the storage
//
// Returns:
//   - *Store: A new storage instance
func NewStore(config *Config) *Store {
	if config == nil {
		config = &Config{}
	}

	shardCount := config.ShardCount
	if shardCount <= 0 {
		shardCount = defaultShardCount
	}
	// Ensure shard count is a power of 2
	// 确保分片数量是2的幂次方
	shardCount = nextPowerOfTwo(shardCount)

	initialCapacity := config.InitialCapacity
	if initialCapacity <= 0 {
		initialCapacity = defaultInitialCapacity
	}

	loadFactor := config.LoadFactor
	if loadFactor <= 0 {
		loadFactor = defaultLoadFactor
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	// Create shards
	// 创建分片
	shards := make([]*shard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = &shard{
			items:       make(map[uint64]*Item, initialCapacity/shardCount),
			initialSize: initialCapacity / shardCount,
			loadFactor:  loadFactor,
			trackAccess: config.TrackAccessTime,
		}
	}

	store := &Store{
		shards:     shards,
		shardCount: uint64(shardCount),
		shardMask:  uint64(shardCount - 1),
		config:     config,
		stats:      &Stats{},
		asyncQueue: make(chan asyncOp, 1024),
		closeChan:  make(chan struct{}),
	}

	// 如果启用异步更新访问时间，则启动后台工作协程
	if config.AsyncAccessUpdate {
		for i := 0; i < runtime.NumCPU(); i++ {
			go store.asyncWorker()
		}
	}

	return store
}

// Get 获取指定键的条目
func (s *Store) Get(key uint64) (*Item, bool) {
	shardIndex := key & s.shardMask
	shard := s.shards[shardIndex]

	startTime := time.Now().UnixNano()

	// 使用读锁获取条目
	shard.RLock()
	item, found := shard.items[key]
	shard.RUnlock()

	// 更新统计信息
	if found {
		atomic.AddUint64(&s.stats.Hits, 1)
		// 如果启用异步更新访问时间，则通过异步队列更新
		if s.config.AsyncAccessUpdate && shard.trackAccess {
			select {
			case s.asyncQueue <- asyncOp{
				op:  opAccessUpdate,
				key: key,
			}:
				// 成功添加到异步队列
			default:
				// 队列已满，丢弃更新
				atomic.AddUint64(&s.stats.AsyncDropCount, 1)
			}
		} else if shard.trackAccess {
			// 同步更新访问时间
			shard.Lock()
			if item, ok := shard.items[key]; ok {
				item.AccessTime = time.Now().UnixNano()
			}
			shard.Unlock()
		}
	} else {
		atomic.AddUint64(&s.stats.Misses, 1)
	}

	// 更新读取延迟统计
	atomic.StoreInt64(&s.stats.ReadLatencyNs, time.Now().UnixNano()-startTime)

	return item, found
}

// Set 设置指定键的条目
func (s *Store) Set(key uint64, value interface{}, size, cost int64, expireAt int64) {
	shardIndex := key & s.shardMask
	shard := s.shards[shardIndex]

	startTime := time.Now().UnixNano()

	item := &Item{
		Key:        key,
		Value:      value,
		Size:       size,
		AccessTime: time.Now().UnixNano(),
		ExpireAt:   expireAt,
		Cost:       cost,
	}

	// 使用写锁设置条目
	shard.Lock()
	oldItem, exists := shard.items[key]
	shard.items[key] = item
	shard.Unlock()

	// 更新统计信息
	if exists {
		atomic.AddUint64(&s.stats.OverwriteCount, 1)
		atomic.AddInt64(&s.stats.BytesSize, size-oldItem.Size)
	} else {
		atomic.AddInt64(&s.stats.ItemCount, 1)
		atomic.AddInt64(&s.stats.BytesSize, size)
	}

	// 更新写入延迟统计
	atomic.StoreInt64(&s.stats.WriteLatencyNs, time.Now().UnixNano()-startTime)
}

// SetBatch 批量设置条目
func (s *Store) SetBatch(items []*Item) {
	// 按分片分组条目
	shardItems := make(map[uint64][]*Item)
	for _, item := range items {
		shardIndex := item.Key & s.shardMask
		shardItems[shardIndex] = append(shardItems[shardIndex], item)
	}

	startTime := time.Now().UnixNano()

	// 对每个分片进行批量设置
	for shardIndex, shardItemList := range shardItems {
		shard := s.shards[shardIndex]
		shard.Lock()
		for _, item := range shardItemList {
			oldItem, exists := shard.items[item.Key]
			shard.items[item.Key] = item

			// 更新统计信息
			if exists {
				atomic.AddUint64(&s.stats.OverwriteCount, 1)
				atomic.AddInt64(&s.stats.BytesSize, item.Size-oldItem.Size)
			} else {
				atomic.AddInt64(&s.stats.ItemCount, 1)
				atomic.AddInt64(&s.stats.BytesSize, item.Size)
			}
		}
		shard.Unlock()
	}

	// 更新写入延迟统计
	atomic.StoreInt64(&s.stats.WriteLatencyNs, time.Now().UnixNano()-startTime)
}

// Delete 删除指定键的条目
func (s *Store) Delete(key uint64) bool {
	shardIndex := key & s.shardMask
	shard := s.shards[shardIndex]

	startTime := time.Now().UnixNano()

	// 使用写锁删除条目
	shard.Lock()
	item, exists := shard.items[key]
	if exists {
		delete(shard.items, key)
	}
	shard.Unlock()

	// 更新统计信息
	if exists {
		atomic.AddInt64(&s.stats.ItemCount, -1)
		atomic.AddInt64(&s.stats.BytesSize, -item.Size)
	}

	// 更新删除延迟统计
	atomic.StoreInt64(&s.stats.DeleteLatencyNs, time.Now().UnixNano()-startTime)

	return exists
}

// DeleteExpired 删除所有过期的条目
func (s *Store) DeleteExpired() int {
	count := 0
	now := time.Now().UnixNano()

	// 遍历所有分片
	for i := uint64(0); i < s.shardCount; i++ {
		shard := s.shards[i]
		var expiredKeys []uint64

		// 使用读锁收集过期的键
		shard.RLock()
		for k, item := range shard.items {
			if item.ExpireAt > 0 && item.ExpireAt <= now {
				expiredKeys = append(expiredKeys, k)
			}
		}
		shard.RUnlock()

		// 如果有过期的键，则使用写锁删除它们
		if len(expiredKeys) > 0 {
			shard.Lock()
			for _, k := range expiredKeys {
				if item, exists := shard.items[k]; exists {
					// 再次检查过期时间，因为在获取写锁的过程中可能已经被其他协程修改
					if item.ExpireAt > 0 && item.ExpireAt <= now {
						delete(shard.items, k)
						atomic.AddInt64(&s.stats.ItemCount, -1)
						atomic.AddInt64(&s.stats.BytesSize, -item.Size)
						count++
					}
				}
			}
			shard.Unlock()
		}
	}

	// 更新过期条目统计
	atomic.AddUint64(&s.stats.ExpiredCount, uint64(count))

	return count
}

// Clear 清空存储
func (s *Store) Clear() {
	for i := uint64(0); i < s.shardCount; i++ {
		shard := s.shards[i]
		shard.Lock()
		shard.items = make(map[uint64]*Item, shard.initialSize)
		shard.Unlock()
	}

	// 重置统计信息
	atomic.StoreInt64(&s.stats.ItemCount, 0)
	atomic.StoreInt64(&s.stats.BytesSize, 0)
}

// Count 返回条目数量
func (s *Store) Count() int64 {
	return atomic.LoadInt64(&s.stats.ItemCount)
}

// Size 返回存储大小（字节）
func (s *Store) Size() int64 {
	return atomic.LoadInt64(&s.stats.BytesSize)
}

// Stats 返回存储统计信息
func (s *Store) Stats() *Stats {
	return &Stats{
		Hits:            atomic.LoadUint64(&s.stats.Hits),
		Misses:          atomic.LoadUint64(&s.stats.Misses),
		Evictions:       atomic.LoadUint64(&s.stats.Evictions),
		Expirations:     atomic.LoadUint64(&s.stats.Expirations),
		ItemCount:       atomic.LoadInt64(&s.stats.ItemCount),
		BytesSize:       atomic.LoadInt64(&s.stats.BytesSize),
		EvictionCost:    atomic.LoadInt64(&s.stats.EvictionCost),
		EvictionCount:   atomic.LoadUint64(&s.stats.EvictionCount),
		ExpiredCount:    atomic.LoadUint64(&s.stats.ExpiredCount),
		ConflictCount:   atomic.LoadUint64(&s.stats.ConflictCount),
		OverwriteCount:  atomic.LoadUint64(&s.stats.OverwriteCount),
		AsyncQueueSize:  int64(len(s.asyncQueue)),
		AsyncDropCount:  atomic.LoadUint64(&s.stats.AsyncDropCount),
		ReadLatencyNs:   atomic.LoadInt64(&s.stats.ReadLatencyNs),
		WriteLatencyNs:  atomic.LoadInt64(&s.stats.WriteLatencyNs),
		DeleteLatencyNs: atomic.LoadInt64(&s.stats.DeleteLatencyNs),
	}
}

// Close 关闭存储
func (s *Store) Close() {
	s.closeOnce.Do(func() {
		close(s.closeChan)
	})
}

// ForEach 遍历所有条目
func (s *Store) ForEach(f func(key uint64, item *Item) bool) {
	for i := uint64(0); i < s.shardCount; i++ {
		shard := s.shards[i]
		shard.RLock()
		for k, v := range shard.items {
			if !f(k, v) {
				shard.RUnlock()
				return
			}
		}
		shard.RUnlock()
	}
}

// Keys 返回所有键
func (s *Store) Keys() []uint64 {
	keys := make([]uint64, 0, s.Count())
	for i := uint64(0); i < s.shardCount; i++ {
		shard := s.shards[i]
		shard.RLock()
		for k := range shard.items {
			keys = append(keys, k)
		}
		shard.RUnlock()
	}
	return keys
}

// asyncWorker 处理异步操作的工作协程
func (s *Store) asyncWorker() {
	for {
		select {
		case op := <-s.asyncQueue:
			switch op.op {
			case opAccessUpdate:
				shardIndex := op.key & s.shardMask
				shard := s.shards[shardIndex]
				shard.Lock()
				if item, ok := shard.items[op.key]; ok {
					item.AccessTime = time.Now().UnixNano()
				}
				shard.Unlock()
			case opSet:
				shardIndex := op.key & s.shardMask
				shard := s.shards[shardIndex]
				shard.Lock()
				oldItem, exists := shard.items[op.key]
				shard.items[op.key] = op.value
				shard.Unlock()

				// 更新统计信息
				if exists {
					atomic.AddUint64(&s.stats.OverwriteCount, 1)
					atomic.AddInt64(&s.stats.BytesSize, op.value.Size-oldItem.Size)
				} else {
					atomic.AddInt64(&s.stats.ItemCount, 1)
					atomic.AddInt64(&s.stats.BytesSize, op.value.Size)
				}

				if op.result != nil {
					op.result <- nil
				}
			case opDelete:
				shardIndex := op.key & s.shardMask
				shard := s.shards[shardIndex]
				shard.Lock()
				item, exists := shard.items[op.key]
				if exists {
					delete(shard.items, op.key)
					atomic.AddInt64(&s.stats.ItemCount, -1)
					atomic.AddInt64(&s.stats.BytesSize, -item.Size)
				}
				shard.Unlock()

				if op.result != nil {
					op.result <- nil
				}
			}
		case <-s.closeChan:
			return
		}
	}
}

// 辅助函数：计算大于等于n的最小2的幂次方
func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}

// 哈希函数，用于计算键的哈希值
func hash(key uint64) uint64 {
	// 使用FNV-1a哈希算法
	h := fnv.New64a()
	// 将uint64转换为字节切片
	b := (*[8]byte)(unsafe.Pointer(&key))[:]
	h.Write(b)
	return h.Sum64()
}
