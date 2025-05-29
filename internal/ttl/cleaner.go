// Package ttl 提供缓存项生命周期管理
package ttl

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/hcache/internal/storage"
)

// Cleaner 提供高效的过期项清理机制
// 通过分片清理和并发处理减少锁竞争
type Cleaner struct {
	store         *storage.Store                      // 存储引用
	cleanInterval time.Duration                       // 清理间隔
	maxCleanItems int                                 // 每次清理的最大项数
	batchSize     int                                 // 每批处理的项数
	concurrency   int                                 // 并发清理的协程数
	closeChan     chan struct{}                       // 关闭信号
	closeOnce     sync.Once                           // 确保只关闭一次
	wg            sync.WaitGroup                      // 等待组
	cleanCount    uint64                              // 清理次数
	expiredCount  uint64                              // 过期项数量
	cleanDuration int64                               // 清理耗时（纳秒）
	onExpired     func(key uint64, value interface{}) // 过期回调函数
}

// CleanerConfig 清理器配置
type CleanerConfig struct {
	// 清理间隔（秒）
	CleanInterval int64

	// 每次清理的最大项数
	MaxCleanItems int

	// 每批处理的项数
	BatchSize int

	// 并发清理的协程数，默认为CPU核心数
	Concurrency int

	// 过期回调函数
	OnExpired func(key uint64, value interface{})
}

// NewCleaner 创建一个新的清理器
func NewCleaner(store *storage.Store, config *CleanerConfig) *Cleaner {
	if config == nil {
		config = &CleanerConfig{}
	}

	// 设置默认值
	cleanInterval := config.CleanInterval
	if cleanInterval <= 0 {
		cleanInterval = defaultCleanInterval
	}

	maxCleanItems := config.MaxCleanItems
	if maxCleanItems <= 0 {
		maxCleanItems = defaultMaxCleanItems
	}

	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	concurrency := config.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	cleaner := &Cleaner{
		store:         store,
		cleanInterval: time.Duration(cleanInterval) * time.Second,
		maxCleanItems: maxCleanItems,
		batchSize:     batchSize,
		concurrency:   concurrency,
		closeChan:     make(chan struct{}),
		onExpired:     config.OnExpired,
	}

	// 启动清理协程
	cleaner.wg.Add(1)
	go cleaner.cleanerLoop()

	return cleaner
}

// cleanerLoop 清理循环，定期清理过期项
func (c *Cleaner) cleanerLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cleanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanExpired()
		case <-c.closeChan:
			return
		}
	}
}

// cleanExpired 清理过期项
func (c *Cleaner) cleanExpired() {
	startTime := time.Now()

	// 获取所有键
	keys := c.store.Keys()

	// 如果没有键，则直接返回
	if len(keys) == 0 {
		return
	}

	// 限制清理的项数
	if len(keys) > c.maxCleanItems {
		keys = keys[:c.maxCleanItems]
	}

	// 计算每个协程处理的键数量
	keysPerGoroutine := (len(keys) + c.concurrency - 1) / c.concurrency
	if keysPerGoroutine < c.batchSize {
		keysPerGoroutine = c.batchSize
	}

	// 并发清理
	var wg sync.WaitGroup
	var expiredCount uint64

	for i := 0; i < c.concurrency && i*keysPerGoroutine < len(keys); i++ {
		wg.Add(1)
		go func(startIdx int) {
			defer wg.Done()

			endIdx := startIdx + keysPerGoroutine
			if endIdx > len(keys) {
				endIdx = len(keys)
			}

			count := c.cleanBatch(keys[startIdx:endIdx])
			atomic.AddUint64(&expiredCount, uint64(count))
		}(i * keysPerGoroutine)
	}

	// 等待所有清理协程完成
	wg.Wait()

	// 更新统计信息
	atomic.AddUint64(&c.cleanCount, 1)
	atomic.AddUint64(&c.expiredCount, expiredCount)
	atomic.StoreInt64(&c.cleanDuration, time.Since(startTime).Nanoseconds())
}

// cleanBatch 批量清理过期项
func (c *Cleaner) cleanBatch(keys []uint64) int {
	now := time.Now().UnixNano()
	count := 0

	for _, key := range keys {
		// 从存储中获取项
		item, found := c.store.Get(key)
		if !found {
			continue
		}

		// 检查是否过期
		if item.ExpireAt > 0 && item.ExpireAt <= now {
			// 删除过期项
			if c.store.Delete(key) {
				count++

				// 如果有过期回调，则调用
				if c.onExpired != nil {
					c.onExpired(key, item.Value)
				}
			}
		}
	}

	return count
}

// Close 关闭清理器
func (c *Cleaner) Close() {
	c.closeOnce.Do(func() {
		close(c.closeChan)
	})
	c.wg.Wait()
}

// GetStats 获取清理器的统计信息
func (c *Cleaner) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"clean_count":     atomic.LoadUint64(&c.cleanCount),
		"expired_count":   atomic.LoadUint64(&c.expiredCount),
		"clean_duration":  time.Duration(atomic.LoadInt64(&c.cleanDuration)).String(),
		"clean_interval":  c.cleanInterval.String(),
		"max_clean_items": c.maxCleanItems,
		"batch_size":      c.batchSize,
		"concurrency":     c.concurrency,
	}
}

// RegisterExpiredCallback 注册过期回调函数
func (c *Cleaner) RegisterExpiredCallback(callback func(key uint64, value interface{})) {
	c.onExpired = callback
}

// ForceClean 强制执行一次清理
func (c *Cleaner) ForceClean() int {
	startTime := time.Now()

	// 获取所有键
	keys := c.store.Keys()

	// 如果没有键，则直接返回
	if len(keys) == 0 {
		return 0
	}

	// 并发清理
	var wg sync.WaitGroup
	var expiredCount uint64

	// 计算每个协程处理的键数量
	keysPerGoroutine := (len(keys) + c.concurrency - 1) / c.concurrency
	if keysPerGoroutine < c.batchSize {
		keysPerGoroutine = c.batchSize
	}

	for i := 0; i < c.concurrency && i*keysPerGoroutine < len(keys); i++ {
		wg.Add(1)
		go func(startIdx int) {
			defer wg.Done()

			endIdx := startIdx + keysPerGoroutine
			if endIdx > len(keys) {
				endIdx = len(keys)
			}

			count := c.cleanBatch(keys[startIdx:endIdx])
			atomic.AddUint64(&expiredCount, uint64(count))
		}(i * keysPerGoroutine)
	}

	// 等待所有清理协程完成
	wg.Wait()

	// 更新统计信息
	atomic.AddUint64(&c.cleanCount, 1)
	atomic.AddUint64(&c.expiredCount, expiredCount)
	atomic.StoreInt64(&c.cleanDuration, time.Since(startTime).Nanoseconds())

	return int(expiredCount)
}

// SetCleanInterval 设置清理间隔
func (c *Cleaner) SetCleanInterval(interval time.Duration) {
	c.cleanInterval = interval
}

// SetMaxCleanItems 设置每次清理的最大项数
func (c *Cleaner) SetMaxCleanItems(maxItems int) {
	c.maxCleanItems = maxItems
}

// SetBatchSize 设置每批处理的项数
func (c *Cleaner) SetBatchSize(batchSize int) {
	c.batchSize = batchSize
}

// SetConcurrency 设置并发清理的协程数
func (c *Cleaner) SetConcurrency(concurrency int) {
	c.concurrency = concurrency
}
