// Package metrics 提供缓存运行时指标采集、统计和输出功能
package metrics

import (
	"math"
	"sync"
	"sync/atomic"
)

// Histogram 延迟直方图，用于统计延迟分布
// 使用原子操作确保高并发安全
type Histogram struct {
	// 桶边界，单位为纳秒
	bucketBounds []int64
	// 桶计数
	bucketCounts []uint64
	// 总计数
	count uint64
	// 最小值
	min int64
	// 最大值
	max int64
	// 总和
	sum int64
	// 互斥锁，用于保护非原子操作
	mu sync.RWMutex
}

// HistogramSnapshot 直方图快照
type HistogramSnapshot struct {
	BucketBounds []int64  `json:"bucket_bounds"`
	BucketCounts []uint64 `json:"bucket_counts"`
	Count        uint64   `json:"count"`
	Min          int64    `json:"min"`
	Max          int64    `json:"max"`
	Sum          int64    `json:"sum"`
	Mean         float64  `json:"mean"`
	P50          int64    `json:"p50"`
	P90          int64    `json:"p90"`
	P99          int64    `json:"p99"`
}

// NewHistogram 创建一个新的直方图
// bucketCount 为桶数量，将自动生成指数分布的桶边界
func NewHistogram(bucketCount int) *Histogram {
	if bucketCount <= 0 {
		bucketCount = 10 // 默认10个桶
	}

	// 创建指数分布的桶边界，从1微秒到10秒
	minLatency := float64(1000)        // 1微秒（纳秒单位）
	maxLatency := float64(10000000000) // 10秒（纳秒单位）

	bucketBounds := make([]int64, bucketCount+1)
	for i := 0; i <= bucketCount; i++ {
		// 使用对数尺度
		power := float64(i) / float64(bucketCount)
		bucketBounds[i] = int64(minLatency * math.Pow(maxLatency/minLatency, power))
	}

	return &Histogram{
		bucketBounds: bucketBounds,
		bucketCounts: make([]uint64, bucketCount+1),
		min:          math.MaxInt64,
		max:          0,
		sum:          0,
		count:        0,
	}
}

// RecordLatency 记录一个延迟值
func (h *Histogram) RecordLatency(latencyNs int64) {
	// 更新最小值、最大值和总和
	h.updateStats(latencyNs)

	// 找到对应的桶并增加计数
	bucketIndex := h.findBucket(latencyNs)
	atomic.AddUint64(&h.bucketCounts[bucketIndex], 1)
	atomic.AddUint64(&h.count, 1)
}

// updateStats 更新统计信息
func (h *Histogram) updateStats(latencyNs int64) {
	// 更新最小值
	for {
		min := atomic.LoadInt64(&h.min)
		if latencyNs >= min {
			break
		}
		if atomic.CompareAndSwapInt64(&h.min, min, latencyNs) {
			break
		}
	}

	// 更新最大值
	for {
		max := atomic.LoadInt64(&h.max)
		if latencyNs <= max {
			break
		}
		if atomic.CompareAndSwapInt64(&h.max, max, latencyNs) {
			break
		}
	}

	// 更新总和
	atomic.AddInt64(&h.sum, latencyNs)
}

// findBucket 找到延迟值对应的桶索引
func (h *Histogram) findBucket(latencyNs int64) int {
	// 二分查找
	i, j := 0, len(h.bucketBounds)-1
	for i < j {
		mid := (i + j) / 2
		if latencyNs > h.bucketBounds[mid] {
			i = mid + 1
		} else {
			j = mid
		}
	}
	return i
}

// Reset 重置直方图
func (h *Histogram) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.bucketCounts {
		atomic.StoreUint64(&h.bucketCounts[i], 0)
	}

	atomic.StoreUint64(&h.count, 0)
	atomic.StoreInt64(&h.min, math.MaxInt64)
	atomic.StoreInt64(&h.max, 0)
	atomic.StoreInt64(&h.sum, 0)
}

// GetSnapshot 获取直方图快照
func (h *Histogram) GetSnapshot() *HistogramSnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := atomic.LoadUint64(&h.count)
	if count == 0 {
		return &HistogramSnapshot{
			BucketBounds: h.bucketBounds,
			BucketCounts: make([]uint64, len(h.bucketCounts)),
			Count:        0,
			Min:          0,
			Max:          0,
			Sum:          0,
			Mean:         0,
			P50:          0,
			P90:          0,
			P99:          0,
		}
	}

	// 复制桶计数
	bucketCounts := make([]uint64, len(h.bucketCounts))
	for i := range h.bucketCounts {
		bucketCounts[i] = atomic.LoadUint64(&h.bucketCounts[i])
	}

	min := atomic.LoadInt64(&h.min)
	max := atomic.LoadInt64(&h.max)
	sum := atomic.LoadInt64(&h.sum)
	mean := float64(sum) / float64(count)

	// 计算百分位数
	p50 := h.calculatePercentile(bucketCounts, 0.5)
	p90 := h.calculatePercentile(bucketCounts, 0.9)
	p99 := h.calculatePercentile(bucketCounts, 0.99)

	return &HistogramSnapshot{
		BucketBounds: h.bucketBounds,
		BucketCounts: bucketCounts,
		Count:        count,
		Min:          min,
		Max:          max,
		Sum:          sum,
		Mean:         mean,
		P50:          p50,
		P90:          p90,
		P99:          p99,
	}
}

// calculatePercentile 计算百分位数
func (h *Histogram) calculatePercentile(bucketCounts []uint64, percentile float64) int64 {
	if percentile < 0 || percentile > 1 {
		return 0
	}

	count := uint64(0)
	for _, c := range bucketCounts {
		count += c
	}

	if count == 0 {
		return 0
	}

	// 计算目标计数
	targetCount := uint64(float64(count) * percentile)

	// 累积计数
	cumulativeCount := uint64(0)

	// 找到目标桶
	for i, c := range bucketCounts {
		cumulativeCount += c
		if cumulativeCount >= targetCount {
			// 如果是最后一个桶，则返回最大值
			if i == len(bucketCounts)-1 {
				return h.bucketBounds[i]
			}

			// 线性插值
			bucketStart := h.bucketBounds[i]
			bucketEnd := h.bucketBounds[i+1]

			// 计算桶内位置
			prevCumulativeCount := cumulativeCount - c
			bucketPosition := float64(targetCount-prevCumulativeCount) / float64(c)

			// 线性插值计算延迟值
			return bucketStart + int64(float64(bucketEnd-bucketStart)*bucketPosition)
		}
	}

	// 如果没有找到，则返回最大值
	return h.bucketBounds[len(h.bucketBounds)-1]
}

// GetBucketCount 获取桶数量
func (h *Histogram) GetBucketCount() int {
	return len(h.bucketCounts)
}

// GetBucketBounds 获取桶边界
func (h *Histogram) GetBucketBounds() []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	bounds := make([]int64, len(h.bucketBounds))
	copy(bounds, h.bucketBounds)
	return bounds
}

// GetBucketCounts 获取桶计数
func (h *Histogram) GetBucketCounts() []uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	counts := make([]uint64, len(h.bucketCounts))
	for i := range h.bucketCounts {
		counts[i] = atomic.LoadUint64(&h.bucketCounts[i])
	}
	return counts
}

// GetCount 获取总计数
func (h *Histogram) GetCount() uint64 {
	return atomic.LoadUint64(&h.count)
}

// GetMin 获取最小值
func (h *Histogram) GetMin() int64 {
	return atomic.LoadInt64(&h.min)
}

// GetMax 获取最大值
func (h *Histogram) GetMax() int64 {
	return atomic.LoadInt64(&h.max)
}

// GetSum 获取总和
func (h *Histogram) GetSum() int64 {
	return atomic.LoadInt64(&h.sum)
}

// GetMean 获取平均值
func (h *Histogram) GetMean() float64 {
	count := atomic.LoadUint64(&h.count)
	if count == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&h.sum)) / float64(count)
}

// GetPercentile 获取指定百分位数
func (h *Histogram) GetPercentile(percentile float64) int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	bucketCounts := make([]uint64, len(h.bucketCounts))
	for i := range h.bucketCounts {
		bucketCounts[i] = atomic.LoadUint64(&h.bucketCounts[i])
	}

	return h.calculatePercentile(bucketCounts, percentile)
}
