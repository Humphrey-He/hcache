// Package eviction 提供缓存淘汰策略实现
package eviction

import (
	"container/list"
	"sync"
	"time"
)

// WTinyLFUPolicy 实现基于W-TinyLFU的淘汰策略
// 结合了窗口缓存和主缓存，使用TinyLFU进行准入控制
type WTinyLFUPolicy struct {
	*BasePolicy
	mu            sync.RWMutex
	windowCache   *LRUCache        // 窗口缓存（最近添加的条目）
	mainCache     *LFUHeapPolicy   // 主缓存（频率较高的条目）
	windowRatio   float64          // 窗口缓存占比
	windowMaxSize int64            // 窗口缓存最大大小
	mainMaxSize   int64            // 主缓存最大大小
	sketch        *FrequencySketch // 频率统计
}

// NewWTinyLFUPolicy 创建一个新的W-TinyLFU淘汰策略
func NewWTinyLFUPolicy(config *Config) *WTinyLFUPolicy {
	windowRatio := 0.01 // 默认窗口缓存占比1%
	if config.WindowRatio > 0 && config.WindowRatio < 1 {
		windowRatio = config.WindowRatio
	}

	windowMaxSize := int64(float64(config.MaxSize) * windowRatio)
	mainMaxSize := config.MaxSize - windowMaxSize

	// 创建窗口缓存配置
	windowConfig := &Config{
		MaxSize:         windowMaxSize,
		MaxItems:        int(float64(config.MaxItems) * windowRatio),
		EnableJanitor:   config.EnableJanitor,
		JanitorInterval: config.JanitorInterval,
	}

	// 创建主缓存配置
	mainConfig := &Config{
		MaxSize:         mainMaxSize,
		MaxItems:        config.MaxItems - windowConfig.MaxItems,
		EnableJanitor:   config.EnableJanitor,
		JanitorInterval: config.JanitorInterval,
	}

	return &WTinyLFUPolicy{
		BasePolicy:    NewBasePolicy(config),
		windowCache:   NewLRUCache(windowConfig),
		mainCache:     NewLFUHeapPolicy(mainConfig),
		windowRatio:   windowRatio,
		windowMaxSize: windowMaxSize,
		mainMaxSize:   mainMaxSize,
		sketch:        NewFrequencySketch(4, 16),
	}
}

// Add 添加一个新的条目
func (w *WTinyLFUPolicy) Add(entry *Entry) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查是否需要淘汰
	needEvict := w.ShouldEvict(entry.Size)

	// 记录访问频率
	w.sketch.Increment(entry.Key)

	// 尝试从主缓存中获取
	if mainEntry := w.mainCache.Remove(entry.Key); mainEntry != nil {
		// 如果在主缓存中找到，则更新后重新添加到主缓存
		updatedEntry := &Entry{
			Key:        entry.Key,
			Value:      entry.Value,
			Size:       entry.Size,
			AccessTime: time.Now().UnixNano(),
			Frequency:  mainEntry.Frequency + 1,
			ExpireAt:   entry.ExpireAt,
		}
		w.mainCache.Add(updatedEntry)
		w.UpdateSize(entry.Size-mainEntry.Size, 0)
		return needEvict
	}

	// 尝试从窗口缓存中获取
	if windowEntry := w.windowCache.Remove(entry.Key); windowEntry != nil {
		// 如果在窗口缓存中找到，则更新后添加到主缓存
		updatedEntry := &Entry{
			Key:        entry.Key,
			Value:      entry.Value,
			Size:       entry.Size,
			AccessTime: time.Now().UnixNano(),
			Frequency:  windowEntry.Frequency + 1,
			ExpireAt:   entry.ExpireAt,
		}
		w.mainCache.Add(updatedEntry)
		w.UpdateSize(entry.Size-windowEntry.Size, 0)
		return needEvict
	}

	// 如果是新条目，则添加到窗口缓存
	newEntry := &Entry{
		Key:        entry.Key,
		Value:      entry.Value,
		Size:       entry.Size,
		AccessTime: time.Now().UnixNano(),
		Frequency:  1,
		ExpireAt:   entry.ExpireAt,
	}

	// 如果窗口缓存已满，则需要进行淘汰
	if w.windowCache.Size()+newEntry.Size > w.windowMaxSize {
		// 淘汰窗口缓存中的一个条目
		victims := w.windowCache.Evict(newEntry.Size)
		if len(victims) > 0 {
			// 获取被淘汰的条目
			victim := victims[0]

			// 获取主缓存中最不常用的条目
			mainVictims := w.mainCache.Evict(0)
			if len(mainVictims) > 0 {
				mainVictim := mainVictims[0]

				// 比较被淘汰的窗口缓存条目和主缓存中最不常用的条目
				// 如果窗口缓存条目的频率更高，则将其添加到主缓存，并淘汰主缓存中的条目
				if w.sketch.Estimate(victim.Key) > w.sketch.Estimate(mainVictim.Key) {
					// 将窗口缓存条目添加到主缓存
					w.mainCache.Add(victim)
				} else {
					// 将主缓存条目重新添加回主缓存
					w.mainCache.Add(mainVictim)
				}
			}
		}
	}

	// 添加新条目到窗口缓存
	w.windowCache.Add(newEntry)
	w.UpdateSize(newEntry.Size, 1)

	return needEvict
}

// Get 获取一个条目，并更新其访问状态
func (w *WTinyLFUPolicy) Get(key uint64) *Entry {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 记录访问频率
	w.sketch.Increment(key)

	// 先尝试从主缓存中获取
	if entry := w.mainCache.Get(key); entry != nil {
		return entry
	}

	// 再尝试从窗口缓存中获取
	if entry := w.windowCache.Get(key); entry != nil {
		// 如果在窗口缓存中找到，则将其提升到主缓存
		w.windowCache.Remove(entry.Key)
		w.mainCache.Add(entry)
		return entry
	}

	return nil
}

// Remove 从缓存中移除一个条目
func (w *WTinyLFUPolicy) Remove(key uint64) *Entry {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 先尝试从主缓存中移除
	if entry := w.mainCache.Remove(key); entry != nil {
		w.UpdateSize(-entry.Size, -1)
		return entry
	}

	// 再尝试从窗口缓存中移除
	if entry := w.windowCache.Remove(key); entry != nil {
		w.UpdateSize(-entry.Size, -1)
		return entry
	}

	return nil
}

// Evict 淘汰一个或多个条目以释放指定大小的空间
func (w *WTinyLFUPolicy) Evict(size int64) []*Entry {
	w.mu.Lock()
	defer w.mu.Unlock()

	var evicted []*Entry
	var evictedSize int64

	// 按比例从窗口缓存和主缓存中淘汰
	windowEvictSize := int64(float64(size) * w.windowRatio)
	mainEvictSize := size - windowEvictSize

	// 从窗口缓存中淘汰
	if windowEvictSize > 0 {
		windowEvicted := w.windowCache.Evict(windowEvictSize)
		evicted = append(evicted, windowEvicted...)
		for _, entry := range windowEvicted {
			evictedSize += entry.Size
		}
	}

	// 从主缓存中淘汰
	if mainEvictSize > 0 {
		mainEvicted := w.mainCache.Evict(mainEvictSize)
		evicted = append(evicted, mainEvicted...)
		for _, entry := range mainEvicted {
			evictedSize += entry.Size
		}
	}

	// 更新缓存大小和条目数
	w.UpdateSize(-evictedSize, -len(evicted))

	return evicted
}

// Clear 清空缓存
func (w *WTinyLFUPolicy) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.windowCache.Clear()
	w.mainCache.Clear()
	w.sketch.Reset()
	w.UpdateSize(-w.Size(), -w.Len())
}

// Close 关闭淘汰策略，释放资源
func (w *WTinyLFUPolicy) Close() {
	w.windowCache.Close()
	w.mainCache.Close()
	w.BasePolicy.Close()
}

// Len 返回缓存中的条目数量
func (w *WTinyLFUPolicy) Len() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.windowCache.Len() + w.mainCache.Len()
}

// Size 返回缓存当前占用的总大小
func (w *WTinyLFUPolicy) Size() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.windowCache.Size() + w.mainCache.Size()
}

// LRUCache 实现基于LRU（Least Recently Used）的淘汰策略
type LRUCache struct {
	*BasePolicy
	mu    sync.RWMutex
	items map[uint64]*list.Element // 键到链表节点的映射
	list  *list.List               // 双向链表，头部是最近使用的，尾部是最久未使用的
}

// NewLRUCache 创建一个新的LRU缓存
func NewLRUCache(config *Config) *LRUCache {
	return &LRUCache{
		BasePolicy: NewBasePolicy(config),
		items:      make(map[uint64]*list.Element),
		list:       list.New(),
	}
}

// Add 添加一个新的条目
func (lru *LRUCache) Add(entry *Entry) bool {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	// 检查是否需要淘汰
	needEvict := lru.ShouldEvict(entry.Size)

	// 如果已存在，则更新
	if elem, ok := lru.items[entry.Key]; ok {
		node := elem.Value.(*Node)
		lru.UpdateSize(entry.Size-node.Size, 0) // 更新大小变化
		node.Value = entry.Value
		node.Size = entry.Size
		node.AccessTime = time.Now().UnixNano()
		node.Frequency = entry.Frequency
		node.ExpireAt = entry.ExpireAt
		lru.list.MoveToFront(elem) // 移动到链表头部
		return needEvict
	}

	// 创建新节点
	node := &Node{
		Key:        entry.Key,
		Value:      entry.Value,
		Size:       entry.Size,
		AccessTime: time.Now().UnixNano(),
		Frequency:  entry.Frequency,
		ExpireAt:   entry.ExpireAt,
	}

	// 添加到链表头部
	elem := lru.list.PushFront(node)
	node.listElem = elem
	lru.items[entry.Key] = elem

	// 更新缓存大小和条目数
	lru.UpdateSize(entry.Size, 1)

	return needEvict
}

// Get 获取一个条目，并更新其访问状态
func (lru *LRUCache) Get(key uint64) *Entry {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if elem, ok := lru.items[key]; ok {
		node := elem.Value.(*Node)

		// 检查是否过期
		if node.IsExpired() {
			lru.removeElement(elem)
			return nil
		}

		// 更新访问时间并移动到链表头部
		node.AccessTime = time.Now().UnixNano()
		node.Frequency++
		lru.list.MoveToFront(elem)

		return node.ToEntry()
	}

	return nil
}

// Remove 从缓存中移除一个条目
func (lru *LRUCache) Remove(key uint64) *Entry {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if elem, ok := lru.items[key]; ok {
		node := elem.Value.(*Node)
		lru.removeElement(elem)

		return node.ToEntry()
	}

	return nil
}

// Evict 淘汰一个或多个条目以释放指定大小的空间
func (lru *LRUCache) Evict(size int64) []*Entry {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	var evicted []*Entry
	var evictedSize int64

	// 先淘汰过期条目
	var expiredElems []*list.Element
	for elem := lru.list.Back(); elem != nil; elem = elem.Prev() {
		node := elem.Value.(*Node)
		if node.IsExpired() {
			expiredElems = append(expiredElems, elem)
		}
	}

	for _, elem := range expiredElems {
		node := elem.Value.(*Node)
		evicted = append(evicted, node.ToEntry())
		evictedSize += node.Size
		lru.removeElement(elem)

		if evictedSize >= size {
			break
		}
	}

	// 如果淘汰过期条目后仍需要淘汰更多，则按LRU策略淘汰
	for evictedSize < size && lru.list.Len() > 0 {
		elem := lru.list.Back()
		node := elem.Value.(*Node)

		evicted = append(evicted, node.ToEntry())
		evictedSize += node.Size
		lru.removeElement(elem)
	}

	// 更新缓存大小和条目数
	lru.UpdateSize(-evictedSize, -len(evicted))

	return evicted
}

// Clear 清空缓存
func (lru *LRUCache) Clear() {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	lru.items = make(map[uint64]*list.Element)
	lru.list.Init()
	lru.UpdateSize(-lru.Size(), -lru.Len())
}

// removeElement 移除一个链表元素
func (lru *LRUCache) removeElement(elem *list.Element) {
	node := elem.Value.(*Node)
	delete(lru.items, node.Key)
	lru.list.Remove(elem)
	lru.UpdateSize(-node.Size, -1)
}

// FrequencySketch 实现Count-Min Sketch算法，用于估计元素的频率
type FrequencySketch struct {
	counters    [][]uint16   // 计数器矩阵
	depth       int          // 哈希函数数量
	width       int          // 每个哈希函数的计数器宽度
	maxCounters int          // 计数器最大值
	seeds       []uint64     // 哈希函数种子
	mutex       sync.RWMutex // 用于并发访问
}

// NewFrequencySketch 创建一个新的频率统计
func NewFrequencySketch(depth, width int) *FrequencySketch {
	if depth <= 0 {
		depth = 4
	}
	if width <= 0 {
		width = 16
	}

	// 初始化计数器矩阵
	counters := make([][]uint16, depth)
	for i := 0; i < depth; i++ {
		counters[i] = make([]uint16, width)
	}

	// 初始化哈希种子
	seeds := []uint64{0x1234567890ABCDEF, 0xFEDCBA0987654321, 0xABCDEF0123456789, 0x0123456789ABCDEF}
	if len(seeds) < depth {
		for i := len(seeds); i < depth; i++ {
			seeds = append(seeds, seeds[i-len(seeds)]+1)
		}
	}

	return &FrequencySketch{
		counters:    counters,
		depth:       depth,
		width:       width,
		maxCounters: 65535, // uint16的最大值
		seeds:       seeds,
	}
}

// Increment 增加一个键的计数
func (fs *FrequencySketch) Increment(key uint64) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// 对每个哈希函数
	for i := 0; i < fs.depth; i++ {
		// 计算哈希值
		hash := fs.hash(key, fs.seeds[i]) % uint64(fs.width)
		// 增加计数器，但不超过最大值
		if fs.counters[i][hash] < uint16(fs.maxCounters) {
			fs.counters[i][hash]++
		}
	}
}

// Estimate 估计一个键的频率
func (fs *FrequencySketch) Estimate(key uint64) uint16 {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	min := uint16(fs.maxCounters)
	// 取所有哈希函数计数的最小值作为估计值
	for i := 0; i < fs.depth; i++ {
		hash := fs.hash(key, fs.seeds[i]) % uint64(fs.width)
		count := fs.counters[i][hash]
		if count < min {
			min = count
		}
	}

	return min
}

// Reset 重置所有计数器
func (fs *FrequencySketch) Reset() {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// 重置所有计数器
	for i := 0; i < fs.depth; i++ {
		for j := 0; j < fs.width; j++ {
			fs.counters[i][j] = 0
		}
	}
}

// hash 计算哈希值
func (fs *FrequencySketch) hash(key, seed uint64) uint64 {
	h := seed
	h ^= key
	h *= 0x100000001b3
	h ^= h >> 32
	return h
}
