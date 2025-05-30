// Package ttl 提供缓存项生命周期管理
// 负责精确控制缓存项的过期时间并进行主动清理
package ttl

import (
	"container/heap"
	"sync"
	"time"

	"github.com/noobtrump/hcache/internal/storage"
)

const (
	// 默认清理间隔时间（秒）
	defaultCleanInterval = 30

	// 默认过期精度（毫秒）
	defaultExpiryPrecision = 500

	// 默认每次清理的最大项数
	defaultMaxCleanItems = 1000

	// 默认是否启用滑动过期
	defaultSlidingExpiration = false
)

// Config TTL管理器配置
type Config struct {
	// 清理间隔（秒）
	CleanInterval int64

	// 过期精度（毫秒）
	ExpiryPrecision int64

	// 每次清理的最大项数
	MaxCleanItems int

	// 是否启用滑动过期
	SlidingExpiration bool

	// 过期回调函数
	OnExpired func(key uint64, value interface{})
}

// Manager TTL管理器
// 负责管理缓存项的生命周期，包括过期检测和清理
type Manager struct {
	store           *storage.Store // 存储引用
	config          *Config        // 配置
	cleanInterval   time.Duration  // 清理间隔
	expiryPrecision time.Duration  // 过期精度
	expiryHeap      *expiryHeap    // 过期堆
	heapMutex       sync.RWMutex   // 堆互斥锁
	closeChan       chan struct{}  // 关闭信号
	closeOnce       sync.Once      // 确保只关闭一次
	wg              sync.WaitGroup // 等待组
}

// expiryItem 表示一个过期项
type expiryItem struct {
	key      uint64 // 键
	expireAt int64  // 过期时间
	index    int    // 在堆中的索引
}

// expiryHeap 实现堆接口，用于高效管理过期项
type expiryHeap []*expiryItem

// Len 返回堆的长度
func (h expiryHeap) Len() int {
	return len(h)
}

// Less 比较两个项的优先级
// 过期时间早的优先级高
func (h expiryHeap) Less(i, j int) bool {
	return h[i].expireAt < h[j].expireAt
}

// Swap 交换两个项
func (h expiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

// Push 向堆中添加一个项
func (h *expiryHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*expiryItem)
	item.index = n
	*h = append(*h, item)
}

// Pop 从堆中弹出一个项
func (h *expiryHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // 避免内存泄漏
	item.index = -1 // 标记为已移除
	*h = old[0 : n-1]
	return item
}

// NewManager 创建一个新的TTL管理器
func NewManager(store *storage.Store, config *Config) *Manager {
	if config == nil {
		config = &Config{}
	}

	// 设置默认值
	cleanInterval := config.CleanInterval
	if cleanInterval <= 0 {
		cleanInterval = defaultCleanInterval
	}

	expiryPrecision := config.ExpiryPrecision
	if expiryPrecision <= 0 {
		expiryPrecision = defaultExpiryPrecision
	}

	maxCleanItems := config.MaxCleanItems
	if maxCleanItems <= 0 {
		maxCleanItems = defaultMaxCleanItems
	}

	// 创建过期堆
	expiryHeap := &expiryHeap{}
	heap.Init(expiryHeap)

	manager := &Manager{
		store:           store,
		config:          config,
		cleanInterval:   time.Duration(cleanInterval) * time.Second,
		expiryPrecision: time.Duration(expiryPrecision) * time.Millisecond,
		expiryHeap:      expiryHeap,
		closeChan:       make(chan struct{}),
	}

	// 启动清理协程
	manager.wg.Add(1)
	go manager.cleanerLoop()

	return manager
}

// cleanerLoop 清理循环，定期清理过期项
func (m *Manager) cleanerLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.cleanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanExpired()
		case <-m.closeChan:
			return
		}
	}
}

// cleanExpired 清理过期项
func (m *Manager) cleanExpired() {
	now := time.Now().UnixNano()
	maxItems := m.config.MaxCleanItems
	count := 0

	// 使用两种清理策略：
	// 1. 基于过期堆的精确清理
	// 2. 基于存储遍历的批量清理

	// 策略1: 基于过期堆的精确清理
	m.heapMutex.Lock()
	for m.expiryHeap.Len() > 0 && count < maxItems {
		// 获取堆顶元素（最早过期的项）
		item := (*m.expiryHeap)[0]

		// 如果还没过期，则退出循环
		if item.expireAt > now {
			break
		}

		// 弹出堆顶元素
		heap.Pop(m.expiryHeap)
		count++

		// 从存储中获取项
		if storeItem, found := m.store.Get(item.key); found {
			// 再次检查过期时间，因为可能已经被更新
			if storeItem.ExpireAt > 0 && storeItem.ExpireAt <= now {
				// 删除过期项
				m.store.Delete(item.key)

				// 如果有过期回调，则调用
				if m.config.OnExpired != nil {
					go m.config.OnExpired(item.key, storeItem.Value)
				}
			} else if storeItem.ExpireAt > 0 {
				// 如果项的过期时间已更新，则重新加入堆
				m.addToExpiryHeap(item.key, storeItem.ExpireAt)
			}
		}
	}
	m.heapMutex.Unlock()

	// 策略2: 基于存储遍历的批量清理
	// 如果堆中没有足够的过期项，则使用存储的DeleteExpired方法
	if count < maxItems {
		m.store.DeleteExpired()
	}
}

// Set 设置项的过期时间
func (m *Manager) Set(key uint64, expireAt int64) {
	if expireAt <= 0 {
		return
	}

	// 添加到过期堆
	m.addToExpiryHeap(key, expireAt)
}

// Extend 延长项的过期时间
func (m *Manager) Extend(key uint64, duration time.Duration) bool {
	// 从存储中获取项
	item, found := m.store.Get(key)
	if !found {
		return false
	}

	// 如果项没有过期时间，则返回
	if item.ExpireAt <= 0 {
		return false
	}

	// 计算新的过期时间
	newExpireAt := item.ExpireAt + int64(duration)

	// 更新存储中的过期时间
	item.ExpireAt = newExpireAt
	m.store.Set(item.Key, item.Value, item.Size, item.Cost, newExpireAt)

	// 更新过期堆
	m.addToExpiryHeap(key, newExpireAt)

	return true
}

// Refresh 刷新项的过期时间
func (m *Manager) Refresh(key uint64, duration time.Duration) bool {
	// 从存储中获取项
	item, found := m.store.Get(key)
	if !found {
		return false
	}

	// 计算新的过期时间
	newExpireAt := time.Now().UnixNano() + int64(duration)

	// 更新存储中的过期时间
	item.ExpireAt = newExpireAt
	m.store.Set(item.Key, item.Value, item.Size, item.Cost, newExpireAt)

	// 更新过期堆
	m.addToExpiryHeap(key, newExpireAt)

	return true
}

// IsExpired 检查项是否已过期
func (m *Manager) IsExpired(key uint64) bool {
	item, found := m.store.Get(key)
	if !found {
		return true
	}

	return item.IsExpired()
}

// TimeToLive 返回项的剩余生存时间
func (m *Manager) TimeToLive(key uint64) time.Duration {
	item, found := m.store.Get(key)
	if !found || item.ExpireAt <= 0 {
		return 0
	}

	now := time.Now().UnixNano()
	if item.ExpireAt <= now {
		return 0
	}

	return time.Duration(item.ExpireAt - now)
}

// ExpireAt 返回项的过期时间
func (m *Manager) ExpireAt(key uint64) int64 {
	item, found := m.store.Get(key)
	if !found {
		return 0
	}

	return item.ExpireAt
}

// Close 关闭TTL管理器
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		close(m.closeChan)
	})
	m.wg.Wait()
}

// addToExpiryHeap 将项添加到过期堆
func (m *Manager) addToExpiryHeap(key uint64, expireAt int64) {
	if expireAt <= 0 {
		return
	}

	m.heapMutex.Lock()
	defer m.heapMutex.Unlock()

	// 创建新的过期项
	item := &expiryItem{
		key:      key,
		expireAt: expireAt,
	}

	// 添加到堆
	heap.Push(m.expiryHeap, item)
}

// removeFromExpiryHeap 从过期堆中移除项
func (m *Manager) removeFromExpiryHeap(key uint64) {
	m.heapMutex.Lock()
	defer m.heapMutex.Unlock()

	// 遍历堆查找项
	for i, item := range *m.expiryHeap {
		if item.key == key {
			// 从堆中移除
			heap.Remove(m.expiryHeap, i)
			break
		}
	}
}

// OnAccess 处理项被访问的事件
// 如果启用了滑动过期，则延长项的过期时间
func (m *Manager) OnAccess(key uint64) {
	// 如果未启用滑动过期，则直接返回
	if !m.config.SlidingExpiration {
		return
	}

	// 从存储中获取项
	item, found := m.store.Get(key)
	if !found || item.ExpireAt <= 0 {
		return
	}

	// 计算新的过期时间
	now := time.Now().UnixNano()
	originalTTL := item.ExpireAt - item.AccessTime
	if originalTTL <= 0 {
		return
	}

	// 更新过期时间
	newExpireAt := now + originalTTL

	// 如果过期时间变化不大，则不更新
	if newExpireAt-item.ExpireAt < int64(m.expiryPrecision) {
		return
	}

	// 更新存储中的过期时间
	item.ExpireAt = newExpireAt
	m.store.Set(item.Key, item.Value, item.Size, item.Cost, newExpireAt)

	// 更新过期堆
	m.addToExpiryHeap(key, newExpireAt)
}

// RegisterExpiredCallback 注册过期回调函数
func (m *Manager) RegisterExpiredCallback(callback func(key uint64, value interface{})) {
	m.config.OnExpired = callback
}

// GetStats 获取TTL管理器的统计信息
func (m *Manager) GetStats() map[string]interface{} {
	m.heapMutex.RLock()
	heapSize := m.expiryHeap.Len()
	m.heapMutex.RUnlock()

	return map[string]interface{}{
		"heap_size":        heapSize,
		"clean_interval":   m.cleanInterval.Seconds(),
		"expiry_precision": m.expiryPrecision.Milliseconds(),
		"sliding_enabled":  m.config.SlidingExpiration,
	}
}
