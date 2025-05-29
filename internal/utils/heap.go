// Package utils 提供HCache内部使用的通用工具函数
package utils

import (
	"container/heap"
	"sync"
)

// MinHeap 是一个通用的最小堆实现
// 支持任意类型，通过比较函数确定元素顺序
type MinHeap[T any] struct {
	items    []T
	lessFunc func(a, b T) bool
	mu       sync.RWMutex
}

// NewMinHeap 创建一个新的最小堆
// lessFunc 是比较函数，当a应该排在b前面时返回true
func NewMinHeap[T any](lessFunc func(a, b T) bool) *MinHeap[T] {
	return &MinHeap[T]{
		items:    make([]T, 0),
		lessFunc: lessFunc,
	}
}

// Len 返回堆的长度
func (h *MinHeap[T]) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.items)
}

// Less 比较两个元素的顺序
func (h *MinHeap[T]) Less(i, j int) bool {
	return h.lessFunc(h.items[i], h.items[j])
}

// Swap 交换两个元素
func (h *MinHeap[T]) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}

// Push 向堆中添加一个元素
func (h *MinHeap[T]) Push(x interface{}) {
	h.items = append(h.items, x.(T))
}

// Pop 从堆中弹出最小的元素
func (h *MinHeap[T]) Pop() interface{} {
	old := h.items
	n := len(old)
	x := old[n-1]
	h.items = old[0 : n-1]
	return x
}

// Add 添加一个元素到堆中
func (h *MinHeap[T]) Add(item T) {
	h.mu.Lock()
	defer h.mu.Unlock()
	heap.Push(h, item)
}

// Remove 从堆中移除指定索引的元素
func (h *MinHeap[T]) Remove(index int) T {
	h.mu.Lock()
	defer h.mu.Unlock()
	return heap.Remove(h, index).(T)
}

// RemoveTop 移除并返回堆顶元素
func (h *MinHeap[T]) RemoveTop() (T, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.items) == 0 {
		var zero T
		return zero, false
	}

	return heap.Pop(h).(T), true
}

// Peek 查看堆顶元素但不移除
func (h *MinHeap[T]) Peek() (T, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.items) == 0 {
		var zero T
		return zero, false
	}

	return h.items[0], true
}

// Items 返回堆中的所有元素（不保证顺序）
func (h *MinHeap[T]) Items() []T {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]T, len(h.items))
	copy(result, h.items)
	return result
}

// Clear 清空堆
func (h *MinHeap[T]) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.items = make([]T, 0)
}

// ExpiryHeap 是一个专门用于管理过期项的堆
// 按过期时间排序，过期时间最早的在堆顶
type ExpiryHeap[T any] struct {
	heap     *MinHeap[ExpiryItem[T]]
	keyIndex map[uint64]int // 键到索引的映射
	mu       sync.RWMutex
}

// ExpiryItem 表示一个带过期时间的项
type ExpiryItem[T any] struct {
	Key      uint64 // 键
	Value    T      // 值
	ExpireAt int64  // 过期时间（纳秒）
	Index    int    // 在堆中的索引
}

// NewExpiryHeap 创建一个新的过期堆
func NewExpiryHeap[T any]() *ExpiryHeap[T] {
	return &ExpiryHeap[T]{
		heap: NewMinHeap(func(a, b ExpiryItem[T]) bool {
			// 按过期时间排序，过期时间相同则按键排序
			if a.ExpireAt == b.ExpireAt {
				return a.Key < b.Key
			}
			return a.ExpireAt < b.ExpireAt
		}),
		keyIndex: make(map[uint64]int),
	}
}

// Add 添加一个项到堆中
// 如果键已存在，则更新值和过期时间
func (h *ExpiryHeap[T]) Add(key uint64, value T, expireAt int64) {
	if expireAt <= 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果键已存在，则更新
	if index, ok := h.keyIndex[key]; ok {
		item := h.heap.items[index]
		item.Value = value
		item.ExpireAt = expireAt
		h.heap.items[index] = item
		heap.Fix(h.heap, index)
		return
	}

	// 添加新项
	item := ExpiryItem[T]{
		Key:      key,
		Value:    value,
		ExpireAt: expireAt,
	}

	h.heap.Add(item)
	itemIndex := len(h.heap.items) - 1
	h.keyIndex[key] = itemIndex

	// 更新索引
	item = h.heap.items[itemIndex]
	item.Index = itemIndex
	h.heap.items[itemIndex] = item
}

// Remove 从堆中移除指定键的项
func (h *ExpiryHeap[T]) Remove(key uint64) (T, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	index, ok := h.keyIndex[key]
	if !ok {
		var zero T
		return zero, false
	}

	item := h.heap.Remove(index)
	delete(h.keyIndex, key)

	// 更新受影响项的索引
	for i, item := range h.heap.items {
		h.keyIndex[item.Key] = i
		itemCopy := item
		itemCopy.Index = i
		h.heap.items[i] = itemCopy
	}

	return item.Value, true
}

// PopExpired 弹出所有已过期的项
// now 是当前时间（纳秒）
func (h *ExpiryHeap[T]) PopExpired(now int64) []ExpiryItem[T] {
	h.mu.Lock()
	defer h.mu.Unlock()

	var expired []ExpiryItem[T]

	for len(h.heap.items) > 0 {
		item, ok := h.heap.Peek()
		if !ok || item.ExpireAt > now {
			break
		}

		// 弹出过期项
		h.heap.RemoveTop()
		delete(h.keyIndex, item.Key)
		expired = append(expired, item)

		// 更新索引
		for i, item := range h.heap.items {
			h.keyIndex[item.Key] = i
			itemCopy := item
			itemCopy.Index = i
			h.heap.items[i] = itemCopy
		}
	}

	return expired
}

// PeekEarliest 查看最早过期的项但不移除
func (h *ExpiryHeap[T]) PeekEarliest() (ExpiryItem[T], bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	item, ok := h.heap.Peek()
	return item, ok
}

// Len 返回堆中项的数量
func (h *ExpiryHeap[T]) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.heap.Len()
}

// Clear 清空堆
func (h *ExpiryHeap[T]) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.heap.Clear()
	h.keyIndex = make(map[uint64]int)
}

// PriorityQueue 是一个通用的优先级队列实现
// 支持任意类型，通过优先级函数确定元素顺序
type PriorityQueue[T any] struct {
	items        []T
	priorityFunc func(T) int64
	mu           sync.RWMutex
}

// NewPriorityQueue 创建一个新的优先级队列
// priorityFunc 是优先级函数，返回值越小优先级越高
func NewPriorityQueue[T any](priorityFunc func(T) int64) *PriorityQueue[T] {
	pq := &PriorityQueue[T]{
		items:        make([]T, 0),
		priorityFunc: priorityFunc,
	}

	heap.Init(&priorityQueueImpl[T]{
		items:        &pq.items,
		priorityFunc: priorityFunc,
	})

	return pq
}

// Add 添加一个元素到队列中
func (pq *PriorityQueue[T]) Add(item T) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	heap.Push(&priorityQueueImpl[T]{
		items:        &pq.items,
		priorityFunc: pq.priorityFunc,
	}, item)
}

// Pop 移除并返回优先级最高的元素
func (pq *PriorityQueue[T]) Pop() (T, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		var zero T
		return zero, false
	}

	return heap.Pop(&priorityQueueImpl[T]{
		items:        &pq.items,
		priorityFunc: pq.priorityFunc,
	}).(T), true
}

// Peek 查看优先级最高的元素但不移除
func (pq *PriorityQueue[T]) Peek() (T, bool) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if len(pq.items) == 0 {
		var zero T
		return zero, false
	}

	return pq.items[0], true
}

// Len 返回队列中元素的数量
func (pq *PriorityQueue[T]) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.items)
}

// Clear 清空队列
func (pq *PriorityQueue[T]) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.items = make([]T, 0)
}

// priorityQueueImpl 是优先级队列的内部实现
type priorityQueueImpl[T any] struct {
	items        *[]T
	priorityFunc func(T) int64
}

func (pq priorityQueueImpl[T]) Len() int {
	return len(*pq.items)
}

func (pq priorityQueueImpl[T]) Less(i, j int) bool {
	return pq.priorityFunc((*pq.items)[i]) < pq.priorityFunc((*pq.items)[j])
}

func (pq priorityQueueImpl[T]) Swap(i, j int) {
	(*pq.items)[i], (*pq.items)[j] = (*pq.items)[j], (*pq.items)[i]
}

func (pq priorityQueueImpl[T]) Push(x interface{}) {
	*pq.items = append(*pq.items, x.(T))
}

func (pq priorityQueueImpl[T]) Pop() interface{} {
	old := *pq.items
	n := len(old)
	x := old[n-1]
	*pq.items = old[0 : n-1]
	return x
}
