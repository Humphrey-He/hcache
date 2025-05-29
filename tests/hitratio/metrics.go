package hitratio

import (
	"fmt"
	"sync"
	"time"
)

// CacheMetrics 收集缓存性能指标
type CacheMetrics struct {
	mu sync.Mutex

	// 基本计数器
	TotalOperations int64
	Hits            int64
	Misses          int64
	Sets            int64
	Deletes         int64
	Evictions       int64

	// 时间指标
	StartTime      time.Time
	EndTime        time.Time
	TotalDuration  time.Duration
	GetLatencySum  time.Duration
	SetLatencySum  time.Duration
	GetLatencyMax  time.Duration
	SetLatencyMax  time.Duration
	GetLatencyHist map[time.Duration]int // 直方图
	SetLatencyHist map[time.Duration]int // 直方图

	// 按键统计
	KeyStats map[string]*KeyMetrics
}

// KeyMetrics 单个键的指标
type KeyMetrics struct {
	Hits   int
	Misses int
	Sets   int
}

// NewCacheMetrics 创建一个新的指标收集器
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{
		StartTime:      time.Now(),
		GetLatencyHist: make(map[time.Duration]int),
		SetLatencyHist: make(map[time.Duration]int),
		KeyStats:       make(map[string]*KeyMetrics),
	}
}

// RecordGet 记录一次 Get 操作
func (m *CacheMetrics) RecordGet(key string, hit bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalOperations++
	if hit {
		m.Hits++
	} else {
		m.Misses++
	}

	// 更新延迟统计
	m.GetLatencySum += latency
	if latency > m.GetLatencyMax {
		m.GetLatencyMax = latency
	}

	// 更新延迟直方图
	bucket := roundDuration(latency)
	m.GetLatencyHist[bucket]++

	// 更新键统计
	km := m.getOrCreateKeyMetrics(key)
	if hit {
		km.Hits++
	} else {
		km.Misses++
	}
}

// RecordSet 记录一次 Set 操作
func (m *CacheMetrics) RecordSet(key string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalOperations++
	m.Sets++

	// 更新延迟统计
	m.SetLatencySum += latency
	if latency > m.SetLatencyMax {
		m.SetLatencyMax = latency
	}

	// 更新延迟直方图
	bucket := roundDuration(latency)
	m.SetLatencyHist[bucket]++

	// 更新键统计
	km := m.getOrCreateKeyMetrics(key)
	km.Sets++
}

// RecordDelete 记录一次 Delete 操作
func (m *CacheMetrics) RecordDelete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalOperations++
	m.Deletes++
}

// RecordEviction 记录一次淘汰
func (m *CacheMetrics) RecordEviction() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Evictions++
}

// Finish 结束指标收集
func (m *CacheMetrics) Finish() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EndTime = time.Now()
	m.TotalDuration = m.EndTime.Sub(m.StartTime)
}

// GetHitRatio 获取命中率
func (m *CacheMetrics) GetHitRatio() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalGets := m.Hits + m.Misses
	if totalGets == 0 {
		return 0
	}
	return float64(m.Hits) / float64(totalGets)
}

// GetAverageGetLatency 获取平均 Get 延迟
func (m *CacheMetrics) GetAverageGetLatency() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalGets := m.Hits + m.Misses
	if totalGets == 0 {
		return 0
	}
	return time.Duration(m.GetLatencySum.Nanoseconds() / int64(totalGets))
}

// GetAverageSetLatency 获取平均 Set 延迟
func (m *CacheMetrics) GetAverageSetLatency() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Sets == 0 {
		return 0
	}
	return time.Duration(m.SetLatencySum.Nanoseconds() / int64(m.Sets))
}

// GetThroughput 获取每秒操作数
func (m *CacheMetrics) GetThroughput() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TotalDuration == 0 {
		return 0
	}
	return float64(m.TotalOperations) / m.TotalDuration.Seconds()
}

// GetTopHotKeys 获取访问最频繁的键
func (m *CacheMetrics) GetTopHotKeys(n int) []KeyStat {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 创建键统计列表
	stats := make([]KeyStat, 0, len(m.KeyStats))
	for key, metrics := range m.KeyStats {
		stats = append(stats, KeyStat{
			Key:    key,
			Hits:   metrics.Hits,
			Misses: metrics.Misses,
			Sets:   metrics.Sets,
			Total:  metrics.Hits + metrics.Misses + metrics.Sets,
		})
	}

	// 按总访问次数排序
	sortKeyStatsByTotal(stats)

	// 返回前 n 个
	if len(stats) > n {
		stats = stats[:n]
	}
	return stats
}

// GetSummary 获取指标摘要
func (m *CacheMetrics) GetSummary() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalGets := m.Hits + m.Misses
	var hitRatio float64
	if totalGets > 0 {
		hitRatio = float64(m.Hits) / float64(totalGets)
	}

	var avgGetLatency, avgSetLatency time.Duration
	if totalGets > 0 {
		avgGetLatency = time.Duration(m.GetLatencySum.Nanoseconds() / int64(totalGets))
	}
	if m.Sets > 0 {
		avgSetLatency = time.Duration(m.SetLatencySum.Nanoseconds() / int64(m.Sets))
	}

	throughput := float64(m.TotalOperations) / m.TotalDuration.Seconds()

	return fmt.Sprintf(`缓存性能指标摘要:
总操作数: %d
总持续时间: %v
吞吐量: %.2f ops/sec

Get 操作:
  总数: %d
  命中: %d
  未命中: %d
  命中率: %.2f%%
  平均延迟: %v
  最大延迟: %v

Set 操作:
  总数: %d
  平均延迟: %v
  最大延迟: %v

其他:
  删除操作: %d
  淘汰次数: %d`,
		m.TotalOperations, m.TotalDuration, throughput,
		totalGets, m.Hits, m.Misses, hitRatio*100, avgGetLatency, m.GetLatencyMax,
		m.Sets, avgSetLatency, m.SetLatencyMax,
		m.Deletes, m.Evictions)
}

// KeyStat 表示键的统计信息
type KeyStat struct {
	Key    string
	Hits   int
	Misses int
	Sets   int
	Total  int
}

// getOrCreateKeyMetrics 获取或创建键指标
func (m *CacheMetrics) getOrCreateKeyMetrics(key string) *KeyMetrics {
	km, exists := m.KeyStats[key]
	if !exists {
		km = &KeyMetrics{}
		m.KeyStats[key] = km
	}
	return km
}

// roundDuration 将持续时间四舍五入到最接近的桶
func roundDuration(d time.Duration) time.Duration {
	// 桶大小（对数尺度）
	buckets := []time.Duration{
		1 * time.Microsecond,
		10 * time.Microsecond,
		100 * time.Microsecond,
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
	}

	for i, bucket := range buckets {
		if d < bucket {
			if i == 0 {
				return bucket
			}
			// 选择最接近的桶
			if d-buckets[i-1] < bucket-d {
				return buckets[i-1]
			}
			return bucket
		}
	}
	return buckets[len(buckets)-1]
}

// sortKeyStatsByTotal 按总访问次数对键统计进行排序
func sortKeyStatsByTotal(stats []KeyStat) {
	// 简单的冒泡排序
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].Total < stats[j].Total {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
}
