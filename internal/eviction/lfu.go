// Package eviction 提供缓存淘汰策略实现
package eviction

import (
	"container/heap"
	"sync"
	"time"
)

// LFUPolicy 实现基于LFU（Least Frequently Used）的淘汰策略
// 淘汰访问频率最低的条目
type LFUPolicy struct {
	*BasePolicy
	mu       sync.RWMutex
	items    map[uint64]*lfuNode // 键到节点的映射
	freqList *freqList           // 频率列表
}

// NewLFUPolicy 创建一个新的LFU淘汰策略
func NewLFUPolicy(config *Config) *LFUPolicy {
	return &LFUPolicy{
		BasePolicy: NewBasePolicy(config),
		items:      make(map[uint64]*lfuNode),
		freqList:   newFreqList(),
	}
}

// Add 添加一个新的条目
func (lfu *LFUPolicy) Add(entry *Entry) bool {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	// 检查是否需要淘汰
	needEvict := lfu.ShouldEvict(entry.Size)

	// 如果已存在，则更新
	if node, ok := lfu.items[entry.Key]; ok {
		lfu.UpdateSize(entry.Size-node.size, 0) // 更新大小变化
		node.value = entry.Value
		node.size = entry.Size
		node.accessTime = time.Now().UnixNano()
		node.expireAt = entry.ExpireAt
		lfu.freqList.access(node) // 更新频率
		return needEvict
	}

	// 创建新节点
	node := &lfuNode{
		key:        entry.Key,
		value:      entry.Value,
		size:       entry.Size,
		accessTime: time.Now().UnixNano(),
		expireAt:   entry.ExpireAt,
	}

	// 添加到频率列表
	lfu.freqList.add(node)
	lfu.items[entry.Key] = node

	// 更新缓存大小和条目数
	lfu.UpdateSize(entry.Size, 1)

	return needEvict
}

// Get 获取一个条目，并更新其访问状态
func (lfu *LFUPolicy) Get(key uint64) *Entry {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	if node, ok := lfu.items[key]; ok {
		// 检查是否过期
		if node.IsExpired() {
			lfu.removeNode(node)
			return nil
		}

		// 更新访问时间和频率
		node.accessTime = time.Now().UnixNano()
		lfu.freqList.access(node)

		return &Entry{
			Key:        node.key,
			Value:      node.value,
			Size:       node.size,
			AccessTime: node.accessTime,
			Frequency:  node.frequency,
			ExpireAt:   node.expireAt,
		}
	}

	return nil
}

// Remove 从缓存中移除一个条目
func (lfu *LFUPolicy) Remove(key uint64) *Entry {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	if node, ok := lfu.items[key]; ok {
		lfu.removeNode(node)

		return &Entry{
			Key:        node.key,
			Value:      node.value,
			Size:       node.size,
			AccessTime: node.accessTime,
			Frequency:  node.frequency,
			ExpireAt:   node.expireAt,
		}
	}

	return nil
}

// Evict 淘汰一个或多个条目以释放指定大小的空间
func (lfu *LFUPolicy) Evict(size int64) []*Entry {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	var evicted []*Entry
	var evictedSize int64

	// 先淘汰过期条目
	for key, node := range lfu.items {
		if node.IsExpired() {
			evicted = append(evicted, &Entry{
				Key:        node.key,
				Value:      node.value,
				Size:       node.size,
				AccessTime: node.accessTime,
				Frequency:  node.frequency,
				ExpireAt:   node.expireAt,
			})

			evictedSize += node.size
			delete(lfu.items, key)
			lfu.freqList.remove(node)

			if evictedSize >= size {
				break
			}
		}
	}

	// 如果淘汰过期条目后仍需要淘汰更多，则按LFU策略淘汰
	for evictedSize < size && len(lfu.items) > 0 {
		// 获取频率最低的节点
		node := lfu.freqList.evict()
		if node == nil {
			break
		}

		evicted = append(evicted, &Entry{
			Key:        node.key,
			Value:      node.value,
			Size:       node.size,
			AccessTime: node.accessTime,
			Frequency:  node.frequency,
			ExpireAt:   node.expireAt,
		})

		evictedSize += node.size
		delete(lfu.items, node.key)
	}

	// 更新缓存大小和条目数
	lfu.UpdateSize(-evictedSize, -len(evicted))

	return evicted
}

// Clear 清空缓存
func (lfu *LFUPolicy) Clear() {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	lfu.items = make(map[uint64]*lfuNode)
	lfu.freqList = newFreqList()
	lfu.UpdateSize(-lfu.Size(), -lfu.Len())
}

// removeNode 移除一个节点
func (lfu *LFUPolicy) removeNode(node *lfuNode) {
	delete(lfu.items, node.key)
	lfu.freqList.remove(node)
	lfu.UpdateSize(-node.size, -1)
}

// lfuNode 表示LFU缓存中的一个节点
type lfuNode struct {
	key        uint64      // 键
	value      interface{} // 值
	size       int64       // 大小
	accessTime int64       // 最后访问时间
	frequency  uint32      // 访问频率
	expireAt   int64       // 过期时间
	prev       *lfuNode    // 前一个节点
	next       *lfuNode    // 后一个节点
	parent     *freqNode   // 所属的频率节点
}

// IsExpired 判断节点是否已过期
func (n *lfuNode) IsExpired() bool {
	if n.expireAt == 0 {
		return false
	}
	return time.Now().UnixNano() > n.expireAt
}

// freqNode 表示具有相同频率的节点集合
type freqNode struct {
	frequency uint32     // 频率
	nodes     []*lfuNode // 具有该频率的节点列表
	prev      *freqNode  // 前一个频率节点
	next      *freqNode  // 后一个频率节点
}

// freqList 表示频率列表，按频率排序
type freqList struct {
	head    *freqNode            // 头节点（最低频率）
	freqMap map[uint32]*freqNode // 频率到节点的映射
}

// newFreqList 创建一个新的频率列表
func newFreqList() *freqList {
	return &freqList{
		freqMap: make(map[uint32]*freqNode),
	}
}

// add 添加一个新节点
func (fl *freqList) add(node *lfuNode) {
	// 新节点的频率为1
	node.frequency = 1

	// 获取或创建频率为1的节点
	freq := fl.getFreqNode(1)

	// 将节点添加到频率节点
	node.parent = freq
	freq.nodes = append(freq.nodes, node)
}

// access 更新节点的访问频率
func (fl *freqList) access(node *lfuNode) {
	// 从原频率节点中移除
	fl.removeFromFreq(node)

	// 增加频率
	node.frequency++

	// 获取或创建新频率的节点
	freq := fl.getFreqNode(node.frequency)

	// 将节点添加到新频率节点
	node.parent = freq
	freq.nodes = append(freq.nodes, node)
}

// remove 移除一个节点
func (fl *freqList) remove(node *lfuNode) {
	fl.removeFromFreq(node)
	node.parent = nil
}

// evict 淘汰一个频率最低的节点
func (fl *freqList) evict() *lfuNode {
	if fl.head == nil || len(fl.head.nodes) == 0 {
		return nil
	}

	// 获取频率最低的节点
	node := fl.head.nodes[0]

	// 从频率节点中移除
	fl.removeFromFreq(node)

	return node
}

// removeFromFreq 从频率节点中移除一个节点
func (fl *freqList) removeFromFreq(node *lfuNode) {
	if node.parent == nil {
		return
	}

	freq := node.parent

	// 从节点列表中移除
	for i, n := range freq.nodes {
		if n == node {
			freq.nodes = append(freq.nodes[:i], freq.nodes[i+1:]...)
			break
		}
	}

	// 如果频率节点为空，则移除
	if len(freq.nodes) == 0 {
		fl.removeFreqNode(freq)
	}
}

// getFreqNode 获取或创建指定频率的节点
func (fl *freqList) getFreqNode(frequency uint32) *freqNode {
	if freq, ok := fl.freqMap[frequency]; ok {
		return freq
	}

	// 创建新的频率节点
	freq := &freqNode{
		frequency: frequency,
		nodes:     make([]*lfuNode, 0),
	}

	// 添加到映射
	fl.freqMap[frequency] = freq

	// 插入到链表中的正确位置
	if fl.head == nil || fl.head.frequency > frequency {
		// 插入到头部
		freq.next = fl.head
		if fl.head != nil {
			fl.head.prev = freq
		}
		fl.head = freq
	} else {
		// 找到正确的插入位置
		current := fl.head
		for current.next != nil && current.next.frequency <= frequency {
			current = current.next
		}

		// 插入到current之后
		freq.next = current.next
		freq.prev = current
		if current.next != nil {
			current.next.prev = freq
		}
		current.next = freq
	}

	return freq
}

// removeFreqNode 移除一个频率节点
func (fl *freqList) removeFreqNode(freq *freqNode) {
	// 从映射中移除
	delete(fl.freqMap, freq.frequency)

	// 从链表中移除
	if freq.prev != nil {
		freq.prev.next = freq.next
	} else {
		fl.head = freq.next
	}

	if freq.next != nil {
		freq.next.prev = freq.prev
	}
}

// LFUHeapPolicy 实现基于最小堆的LFU淘汰策略
// 相比链表实现，堆实现在大规模数据下有更好的性能
type LFUHeapPolicy struct {
	*BasePolicy
	mu    sync.RWMutex
	items map[uint64]*lfuHeapItem // 键到堆项的映射
	heap  lfuHeap                 // 最小堆
}

// NewLFUHeapPolicy 创建一个新的基于堆的LFU淘汰策略
func NewLFUHeapPolicy(config *Config) *LFUHeapPolicy {
	return &LFUHeapPolicy{
		BasePolicy: NewBasePolicy(config),
		items:      make(map[uint64]*lfuHeapItem),
		heap:       make(lfuHeap, 0),
	}
}

// Add 添加一个新的条目
func (lfu *LFUHeapPolicy) Add(entry *Entry) bool {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	// 检查是否需要淘汰
	needEvict := lfu.ShouldEvict(entry.Size)

	// 如果已存在，则更新
	if item, ok := lfu.items[entry.Key]; ok {
		lfu.UpdateSize(entry.Size-item.size, 0) // 更新大小变化
		item.value = entry.Value
		item.size = entry.Size
		item.accessTime = time.Now().UnixNano()
		item.expireAt = entry.ExpireAt
		item.frequency++
		heap.Fix(&lfu.heap, item.index)
		return needEvict
	}

	// 创建新项
	item := &lfuHeapItem{
		key:        entry.Key,
		value:      entry.Value,
		size:       entry.Size,
		frequency:  1,
		accessTime: time.Now().UnixNano(),
		expireAt:   entry.ExpireAt,
	}

	// 添加到堆和映射
	heap.Push(&lfu.heap, item)
	lfu.items[entry.Key] = item

	// 更新缓存大小和条目数
	lfu.UpdateSize(entry.Size, 1)

	return needEvict
}

// Get 获取一个条目，并更新其访问状态
func (lfu *LFUHeapPolicy) Get(key uint64) *Entry {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	if item, ok := lfu.items[key]; ok {
		// 检查是否过期
		if item.IsExpired() {
			lfu.removeItem(item)
			return nil
		}

		// 更新访问时间和频率
		item.accessTime = time.Now().UnixNano()
		item.frequency++
		heap.Fix(&lfu.heap, item.index)

		return &Entry{
			Key:        item.key,
			Value:      item.value,
			Size:       item.size,
			AccessTime: item.accessTime,
			Frequency:  item.frequency,
			ExpireAt:   item.expireAt,
		}
	}

	return nil
}

// Remove 从缓存中移除一个条目
func (lfu *LFUHeapPolicy) Remove(key uint64) *Entry {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	if item, ok := lfu.items[key]; ok {
		lfu.removeItem(item)

		return &Entry{
			Key:        item.key,
			Value:      item.value,
			Size:       item.size,
			AccessTime: item.accessTime,
			Frequency:  item.frequency,
			ExpireAt:   item.expireAt,
		}
	}

	return nil
}

// Evict 淘汰一个或多个条目以释放指定大小的空间
func (lfu *LFUHeapPolicy) Evict(size int64) []*Entry {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	var evicted []*Entry
	var evictedSize int64

	// 先淘汰过期条目
	var expiredItems []*lfuHeapItem
	for _, item := range lfu.items {
		if item.IsExpired() {
			expiredItems = append(expiredItems, item)
		}
	}

	for _, item := range expiredItems {
		evicted = append(evicted, &Entry{
			Key:        item.key,
			Value:      item.value,
			Size:       item.size,
			AccessTime: item.accessTime,
			Frequency:  item.frequency,
			ExpireAt:   item.expireAt,
		})

		evictedSize += item.size
		lfu.removeItem(item)

		if evictedSize >= size {
			break
		}
	}

	// 如果淘汰过期条目后仍需要淘汰更多，则按LFU策略淘汰
	for evictedSize < size && lfu.heap.Len() > 0 {
		item := heap.Pop(&lfu.heap).(*lfuHeapItem)
		delete(lfu.items, item.key)

		evicted = append(evicted, &Entry{
			Key:        item.key,
			Value:      item.value,
			Size:       item.size,
			AccessTime: item.accessTime,
			Frequency:  item.frequency,
			ExpireAt:   item.expireAt,
		})

		evictedSize += item.size
	}

	// 更新缓存大小和条目数
	lfu.UpdateSize(-evictedSize, -len(evicted))

	return evicted
}

// Clear 清空缓存
func (lfu *LFUHeapPolicy) Clear() {
	lfu.mu.Lock()
	defer lfu.mu.Unlock()

	lfu.items = make(map[uint64]*lfuHeapItem)
	lfu.heap = make(lfuHeap, 0)
	lfu.UpdateSize(-lfu.Size(), -lfu.Len())
}

// removeItem 移除一个堆项
func (lfu *LFUHeapPolicy) removeItem(item *lfuHeapItem) {
	delete(lfu.items, item.key)
	heap.Remove(&lfu.heap, item.index)
	lfu.UpdateSize(-item.size, -1)
}

// lfuHeapItem 表示LFU堆中的一个项
type lfuHeapItem struct {
	key        uint64      // 键
	value      interface{} // 值
	size       int64       // 大小
	frequency  uint32      // 访问频率
	accessTime int64       // 最后访问时间
	expireAt   int64       // 过期时间
	index      int         // 在堆中的索引
}

// IsExpired 判断项是否已过期
func (item *lfuHeapItem) IsExpired() bool {
	if item.expireAt == 0 {
		return false
	}
	return time.Now().UnixNano() > item.expireAt
}

// lfuHeap 实现堆接口
type lfuHeap []*lfuHeapItem

// Len 返回堆的长度
func (h lfuHeap) Len() int {
	return len(h)
}

// Less 比较两个项的优先级
// 优先级：频率低 > 最后访问时间早
func (h lfuHeap) Less(i, j int) bool {
	if h[i].frequency == h[j].frequency {
		return h[i].accessTime < h[j].accessTime
	}
	return h[i].frequency < h[j].frequency
}

// Swap 交换两个项
func (h lfuHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

// Push 向堆中添加一个项
func (h *lfuHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*lfuHeapItem)
	item.index = n
	*h = append(*h, item)
}

// Pop 从堆中弹出一个项
func (h *lfuHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]
	return item
}
