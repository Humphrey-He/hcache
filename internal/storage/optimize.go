// Package storage 提供高性能的分片存储实现
package storage

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Optimizer 存储优化器
// 用于优化内存使用和性能，包括内存回收、分片重平衡等
type Optimizer struct {
	store            *Store         // 存储引用
	interval         time.Duration  // 优化间隔
	maxItems         int            // 每次优化的最大项数
	closeChan        chan struct{}  // 关闭信号
	closeOnce        sync.Once      // 确保只关闭一次
	wg               sync.WaitGroup // 等待组
	optimizeCount    uint64         // 优化次数
	removedCount     uint64         // 移除项数量
	optimizeDuration int64          // 优化耗时（纳秒）
	memoryLimit      int64          // 内存限制（字节）
	costLimit        int64          // 成本限制
	sampleRatio      float64        // 采样比例
}

// OptimizerConfig 优化器配置
type OptimizerConfig struct {
	// 优化间隔（秒）
	Interval int64

	// 每次优化的最大项数
	MaxItems int

	// 内存限制（字节）
	MemoryLimit int64

	// 成本限制
	CostLimit int64

	// 采样比例（0-1）
	SampleRatio float64
}

// NewOptimizer 创建一个新的优化器
func NewOptimizer(store *Store, config *OptimizerConfig) *Optimizer {
	if config == nil {
		config = &OptimizerConfig{}
	}

	// 设置默认值
	interval := config.Interval
	if interval <= 0 {
		interval = 300 // 默认5分钟
	}

	maxItems := config.MaxItems
	if maxItems <= 0 {
		maxItems = 1000
	}

	sampleRatio := config.SampleRatio
	if sampleRatio <= 0 || sampleRatio > 1 {
		sampleRatio = 0.1 // 默认采样10%
	}

	optimizer := &Optimizer{
		store:       store,
		interval:    time.Duration(interval) * time.Second,
		maxItems:    maxItems,
		closeChan:   make(chan struct{}),
		memoryLimit: config.MemoryLimit,
		costLimit:   config.CostLimit,
		sampleRatio: sampleRatio,
	}

	// 启动优化协程
	optimizer.wg.Add(1)
	go optimizer.optimizeLoop()

	return optimizer
}

// optimizeLoop 优化循环，定期优化存储
func (o *Optimizer) optimizeLoop() {
	defer o.wg.Done()

	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			o.optimize()
		case <-o.closeChan:
			return
		}
	}
}

// optimize 执行优化
func (o *Optimizer) optimize() {
	startTime := time.Now()

	// 执行多种优化策略
	removedCount := 0

	// 1. 内存限制优化
	if o.memoryLimit > 0 && o.store.Size() > o.memoryLimit {
		count := o.optimizeMemory()
		removedCount += count
	}

	// 2. 成本限制优化
	if o.costLimit > 0 {
		count := o.optimizeCost()
		removedCount += count
	}

	// 3. 分片重平衡（只在低负载时执行）
	if runtime.NumGoroutine() < 100 { // 简单的低负载判断
		o.rebalanceShards()
	}

	// 4. 空闲内存回收
	if removedCount > 0 {
		runtime.GC()
	}

	// 更新统计信息
	atomic.AddUint64(&o.optimizeCount, 1)
	atomic.AddUint64(&o.removedCount, uint64(removedCount))
	atomic.StoreInt64(&o.optimizeDuration, time.Since(startTime).Nanoseconds())
}

// optimizeMemory 优化内存使用
// 当内存使用超过限制时，淘汰部分项
func (o *Optimizer) optimizeMemory() int {
	// 计算需要释放的内存大小
	currentSize := o.store.Size()
	if currentSize <= o.memoryLimit {
		return 0
	}

	needFree := currentSize - o.memoryLimit
	count := 0

	// 获取所有键（采样）
	allKeys := o.store.Keys()
	if len(allKeys) == 0 {
		return 0
	}

	// 采样
	sampleSize := int(float64(len(allKeys)) * o.sampleRatio)
	if sampleSize > o.maxItems {
		sampleSize = o.maxItems
	}
	if sampleSize <= 0 {
		sampleSize = 1
	}

	// 随机选择键
	keys := make([]uint64, sampleSize)
	for i := 0; i < sampleSize; i++ {
		keys[i] = allKeys[i*len(allKeys)/sampleSize]
	}

	// 按访问时间排序（从旧到新）
	type keyWithTime struct {
		key        uint64
		accessTime int64
		size       int64
	}
	items := make([]keyWithTime, 0, len(keys))

	for _, key := range keys {
		item, found := o.store.Get(key)
		if !found {
			continue
		}
		items = append(items, keyWithTime{
			key:        key,
			accessTime: item.AccessTime,
			size:       item.Size,
		})
	}

	// 按访问时间排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].accessTime < items[j].accessTime
	})

	// 淘汰最旧的项，直到释放足够的内存
	var freedSize int64
	for _, item := range items {
		if o.store.Delete(item.key) {
			freedSize += item.size
			count++
		}
		if freedSize >= needFree {
			break
		}
	}

	return count
}

// optimizeCost 优化成本
// 当总成本超过限制时，淘汰部分项
func (o *Optimizer) optimizeCost() int {
	if o.costLimit <= 0 {
		return 0
	}

	// 计算当前总成本
	var totalCost int64
	o.store.ForEach(func(key uint64, item *Item) bool {
		totalCost += item.Cost
		return true
	})

	if totalCost <= o.costLimit {
		return 0
	}

	needFree := totalCost - o.costLimit
	count := 0

	// 获取所有键（采样）
	allKeys := o.store.Keys()
	if len(allKeys) == 0 {
		return 0
	}

	// 采样
	sampleSize := int(float64(len(allKeys)) * o.sampleRatio)
	if sampleSize > o.maxItems {
		sampleSize = o.maxItems
	}
	if sampleSize <= 0 {
		sampleSize = 1
	}

	// 随机选择键
	keys := make([]uint64, sampleSize)
	for i := 0; i < sampleSize; i++ {
		keys[i] = allKeys[i*len(allKeys)/sampleSize]
	}

	// 按成本排序（从低到高）
	type keyWithCost struct {
		key  uint64
		cost int64
	}
	items := make([]keyWithCost, 0, len(keys))

	for _, key := range keys {
		item, found := o.store.Get(key)
		if !found {
			continue
		}
		items = append(items, keyWithCost{
			key:  key,
			cost: item.Cost,
		})
	}

	// 按成本排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].cost < items[j].cost
	})

	// 淘汰成本最低的项，直到释放足够的成本
	var freedCost int64
	for _, item := range items {
		storeItem, found := o.store.Get(item.key)
		if !found {
			continue
		}
		if o.store.Delete(item.key) {
			freedCost += storeItem.Cost
			count++
		}
		if freedCost >= needFree {
			break
		}
	}

	return count
}

// rebalanceShards 重平衡分片
// 尝试使各个分片的项数更均衡
func (o *Optimizer) rebalanceShards() {
	// 计算每个分片的项数
	shardCounts := make([]int, o.store.shardCount)
	o.store.ForEach(func(key uint64, item *Item) bool {
		shardIndex := key & o.store.shardMask
		shardCounts[shardIndex]++
		return true
	})

	// 计算平均项数
	var totalItems int
	for _, count := range shardCounts {
		totalItems += count
	}
	avgItems := totalItems / len(shardCounts)

	// 找出项数最多和最少的分片
	maxIndex := 0
	minIndex := 0
	for i, count := range shardCounts {
		if count > shardCounts[maxIndex] {
			maxIndex = i
		}
		if count < shardCounts[minIndex] {
			minIndex = i
		}
	}

	// 如果最大分片的项数超过平均值的1.5倍，且最小分片的项数低于平均值的0.5倍，则进行重平衡
	if shardCounts[maxIndex] > avgItems*3/2 && shardCounts[minIndex] < avgItems/2 {
		// 从最大分片中移动一些项到最小分片
		// 注意：这里只是示例，实际实现可能需要更复杂的逻辑
		// 在生产环境中，应该谨慎使用这种操作，因为它可能会导致大量的内存分配和复制
		// 这里仅作为一个概念性的实现
		moveCount := (shardCounts[maxIndex] - shardCounts[minIndex]) / 2
		if moveCount > 0 {
			// 获取最大分片中的所有键
			var maxShardKeys []uint64
			o.store.ForEach(func(key uint64, item *Item) bool {
				shardIndex := key & o.store.shardMask
				if shardIndex == uint64(maxIndex) {
					maxShardKeys = append(maxShardKeys, key)
				}
				return true
			})

			// 移动部分键到最小分片
			// 注意：这里的实现是简化的，实际上应该考虑更多因素
			for i := 0; i < moveCount && i < len(maxShardKeys); i++ {
				key := maxShardKeys[i]
				item, found := o.store.Get(key)
				if !found {
					continue
				}
				// 删除原项
				o.store.Delete(key)
				// 创建新键（修改哈希值使其落在目标分片）
				newKey := (key & ^o.store.shardMask) | uint64(minIndex)
				o.store.Set(newKey, item.Value, item.Size, item.Cost, item.ExpireAt)
			}
		}
	}
}

// Close 关闭优化器
func (o *Optimizer) Close() {
	o.closeOnce.Do(func() {
		close(o.closeChan)
	})
	o.wg.Wait()
}

// GetStats 获取优化器的统计信息
func (o *Optimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"optimize_count":    atomic.LoadUint64(&o.optimizeCount),
		"removed_count":     atomic.LoadUint64(&o.removedCount),
		"optimize_duration": time.Duration(atomic.LoadInt64(&o.optimizeDuration)).String(),
		"interval":          o.interval.String(),
		"max_items":         o.maxItems,
		"memory_limit":      o.memoryLimit,
		"cost_limit":        o.costLimit,
		"sample_ratio":      o.sampleRatio,
	}
}

// ForceOptimize 强制执行一次优化
func (o *Optimizer) ForceOptimize() int {
	startTime := time.Now()

	// 执行优化
	removedCount := 0

	// 1. 内存限制优化
	if o.memoryLimit > 0 && o.store.Size() > o.memoryLimit {
		count := o.optimizeMemory()
		removedCount += count
	}

	// 2. 成本限制优化
	if o.costLimit > 0 {
		count := o.optimizeCost()
		removedCount += count
	}

	// 3. 分片重平衡
	o.rebalanceShards()

	// 4. 空闲内存回收
	if removedCount > 0 {
		runtime.GC()
	}

	// 更新统计信息
	atomic.AddUint64(&o.optimizeCount, 1)
	atomic.AddUint64(&o.removedCount, uint64(removedCount))
	atomic.StoreInt64(&o.optimizeDuration, time.Since(startTime).Nanoseconds())

	return removedCount
}

// SetInterval 设置优化间隔
func (o *Optimizer) SetInterval(interval time.Duration) {
	o.interval = interval
}

// SetMaxItems 设置每次优化的最大项数
func (o *Optimizer) SetMaxItems(maxItems int) {
	o.maxItems = maxItems
}

// SetMemoryLimit 设置内存限制
func (o *Optimizer) SetMemoryLimit(limit int64) {
	o.memoryLimit = limit
}

// SetCostLimit 设置成本限制
func (o *Optimizer) SetCostLimit(limit int64) {
	o.costLimit = limit
}

// SetSampleRatio 设置采样比例
func (o *Optimizer) SetSampleRatio(ratio float64) {
	if ratio > 0 && ratio <= 1 {
		o.sampleRatio = ratio
	}
}
