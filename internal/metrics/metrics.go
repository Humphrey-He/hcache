// Package metrics provides cache runtime metrics collection, statistics, and reporting.
// Package metrics 提供缓存运行时指标采集、统计和输出功能。
//
// This package implements high-performance metrics collection with minimal impact on the main
// business path. It supports different levels of metrics collection, from basic hit/miss ratios
// to detailed latency distributions and shard-level statistics. The metrics are designed to be
// collected atomically to ensure thread safety in high-concurrency environments.
//
// 本包实现了高性能的指标收集，对主业务路径的影响最小。它支持不同级别的指标收集，从基本的
// 命中/未命中率到详细的延迟分布和分片级统计。这些指标被设计为原子收集，以确保在高并发环境中
// 的线程安全。
package metrics

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// Level defines the metrics collection level.
// Level 定义指标采集级别。
type Level int

const (
	// Disabled means metrics collection is turned off.
	// Disabled 表示禁用指标采集。
	Disabled Level = iota

	// Basic enables collection of essential metrics like hit ratio and capacity.
	// Basic 启用基础指标采集（命中率、容量等）。
	Basic

	// Detailed enables collection of comprehensive metrics including latency distribution and shard statistics.
	// Detailed 启用详细指标采集（包括延迟分布、分片统计等）。
	Detailed
)

// Metrics is a cache metrics collector.
// It uses atomic operations to ensure thread safety in high-concurrency environments.
//
// Metrics 是缓存指标收集器。
// 使用原子操作确保高并发环境下的线程安全。
type Metrics struct {
	// Collection level
	// 采集级别
	level Level

	// Hit ratio related metrics
	// 缓存命中率相关指标
	hits     uint64 // Hit count / 命中次数
	misses   uint64 // Miss count / 未命中次数
	hitRatio uint64 // Hit ratio * 10000 (4 decimal places) / 命中率 * 10000（保留4位小数）

	// Write behavior metrics
	// 写入行为相关指标
	sets       uint64 // Set operations count / 设置次数
	updates    uint64 // Update operations count / 更新次数
	overwrites uint64 // Overwrite operations count / 覆盖次数
	rejects    uint64 // Rejected operations count / 拒绝次数

	// Eviction metrics
	// 淘汰行为相关指标
	evictions       uint64 // Eviction count / 淘汰次数
	expired         uint64 // Expired items count / 过期次数
	manuallyDeleted uint64 // Manually deleted items count / 手动删除次数

	// Performance metrics
	// 性能指标
	getLatencySum    uint64 // Sum of get operation latencies (ns) / 读取延迟总和（纳秒）
	getCount         uint64 // Count of get operations / 读取次数
	setLatencySum    uint64 // Sum of set operation latencies (ns) / 写入延迟总和（纳秒）
	setCount         uint64 // Count of set operations / 写入次数
	deleteLatencySum uint64 // Sum of delete operation latencies (ns) / 删除延迟总和（纳秒）
	deleteCount      uint64 // Count of delete operations / 删除次数

	// Capacity pressure metrics
	// 容量压力相关指标
	entryCount  int64   // Number of entries in cache / 条目数量
	memoryUsage int64   // Memory usage in bytes / 内存使用量（字节）
	shardUsage  []int64 // Per-shard usage statistics / 分片使用情况

	// Latency histogram
	// 延迟直方图
	latencyHistogram *Histogram

	// Shard-level metrics
	// 分片级指标
	shardMetrics       []*ShardMetrics
	enableShardMetrics bool

	// Last update timestamp
	// 最后更新时间
	lastUpdated int64

	// Mutex for protecting non-atomic operations
	// 互斥锁，用于保护非原子操作
	mu sync.RWMutex
}

// ShardMetrics contains metrics for an individual cache shard.
// ShardMetrics 包含单个缓存分片的指标。
type ShardMetrics struct {
	ShardID     int    // Shard identifier / 分片ID
	ItemCount   int64  // Number of items in the shard / 条目数量
	MemoryUsage int64  // Memory usage of the shard in bytes / 内存使用量（字节）
	Hits        uint64 // Hit count for this shard / 命中次数
	Misses      uint64 // Miss count for this shard / 未命中次数
	Evictions   uint64 // Eviction count for this shard / 淘汰次数
	Conflicts   uint64 // Hash conflict count for this shard / 冲突次数
}

// Config defines metrics configuration options.
// Config 定义指标配置选项。
type Config struct {
	// Level determines the detail level of metrics collection
	// Level 指定指标采集的详细程度
	Level Level

	// EnableShardMetrics enables per-shard metrics collection
	// EnableShardMetrics 启用分片级指标收集
	EnableShardMetrics bool

	// ShardCount specifies the number of shards in the cache
	// ShardCount 指定缓存中的分片数量
	ShardCount int

	// EnableLatencyHistogram enables latency histogram collection
	// EnableLatencyHistogram 启用延迟直方图收集
	EnableLatencyHistogram bool

	// HistogramBuckets specifies the number of buckets in the latency histogram
	// HistogramBuckets 指定延迟直方图中的桶数量
	HistogramBuckets int
}

// New creates a new metrics collector.
//
// New 创建一个新的指标收集器。
//
// Parameters:
//   - config: Configuration options for the metrics collector
//
// Returns:
//   - *Metrics: A new metrics collector instance
func New(config *Config) *Metrics {
	if config == nil {
		config = &Config{
			Level:                  Basic,
			EnableShardMetrics:     false,
			ShardCount:             0,
			EnableLatencyHistogram: false,
			HistogramBuckets:       0,
		}
	}

	m := &Metrics{
		level:              config.Level,
		enableShardMetrics: config.EnableShardMetrics,
		lastUpdated:        time.Now().UnixNano(),
	}

	// Initialize shard metrics if enabled
	// 初始化分片指标（如果启用）
	if config.EnableShardMetrics && config.ShardCount > 0 {
		m.shardMetrics = make([]*ShardMetrics, config.ShardCount)
		m.shardUsage = make([]int64, config.ShardCount)
		for i := 0; i < config.ShardCount; i++ {
			m.shardMetrics[i] = &ShardMetrics{ShardID: i}
		}
	}

	// Initialize latency histogram if enabled
	// 初始化延迟直方图（如果启用）
	if config.EnableLatencyHistogram && config.HistogramBuckets > 0 {
		m.latencyHistogram = NewHistogram(config.HistogramBuckets)
	}

	return m
}

// RecordHit records a cache hit.
//
// RecordHit 记录缓存命中。
func (m *Metrics) RecordHit() {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.hits, 1)
	m.updateHitRatio()
}

// RecordHitWithShard records a cache hit for a specific shard.
//
// RecordHitWithShard 记录特定分片的缓存命中。
//
// Parameters:
//   - shardID: The ID of the shard where the hit occurred
func (m *Metrics) RecordHitWithShard(shardID int) {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.hits, 1)

	if m.enableShardMetrics && shardID >= 0 && shardID < len(m.shardMetrics) {
		atomic.AddUint64(&m.shardMetrics[shardID].Hits, 1)
	}

	m.updateHitRatio()
}

// RecordMiss records a cache miss.
//
// RecordMiss 记录缓存未命中。
func (m *Metrics) RecordMiss() {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.misses, 1)
	m.updateHitRatio()
}

// RecordMissWithShard records a cache miss for a specific shard.
//
// RecordMissWithShard 记录特定分片的缓存未命中。
//
// Parameters:
//   - shardID: The ID of the shard where the miss occurred
func (m *Metrics) RecordMissWithShard(shardID int) {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.misses, 1)

	if m.enableShardMetrics && shardID >= 0 && shardID < len(m.shardMetrics) {
		atomic.AddUint64(&m.shardMetrics[shardID].Misses, 1)
	}

	m.updateHitRatio()
}

// RecordSet records a cache set operation.
//
// RecordSet 记录缓存设置操作。
func (m *Metrics) RecordSet() {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.sets, 1)
}

// RecordUpdate records a cache update operation.
//
// RecordUpdate 记录缓存更新操作。
func (m *Metrics) RecordUpdate() {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.updates, 1)
}

// RecordOverwrite records a cache overwrite operation.
//
// RecordOverwrite 记录缓存覆盖操作。
func (m *Metrics) RecordOverwrite() {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.overwrites, 1)
}

// RecordReject 记录缓存拒绝
func (m *Metrics) RecordReject() {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.rejects, 1)
}

// RecordEviction 记录缓存淘汰
func (m *Metrics) RecordEviction() {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.evictions, 1)
}

// RecordEvictionWithShard 记录特定分片的缓存淘汰
func (m *Metrics) RecordEvictionWithShard(shardID int) {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.evictions, 1)

	if m.enableShardMetrics && shardID >= 0 && shardID < len(m.shardMetrics) {
		atomic.AddUint64(&m.shardMetrics[shardID].Evictions, 1)
	}
}

// RecordExpired 记录缓存过期
func (m *Metrics) RecordExpired(count int) {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.expired, uint64(count))
}

// RecordManualDelete 记录手动删除
func (m *Metrics) RecordManualDelete() {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.manuallyDeleted, 1)
}

// RecordGetLatency 记录读取延迟
func (m *Metrics) RecordGetLatency(latencyNs int64) {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.getLatencySum, uint64(latencyNs))
	atomic.AddUint64(&m.getCount, 1)

	if m.latencyHistogram != nil && m.level == Detailed {
		m.latencyHistogram.RecordLatency(latencyNs)
	}
}

// RecordSetLatency 记录写入延迟
func (m *Metrics) RecordSetLatency(latencyNs int64) {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.setLatencySum, uint64(latencyNs))
	atomic.AddUint64(&m.setCount, 1)

	if m.latencyHistogram != nil && m.level == Detailed {
		m.latencyHistogram.RecordLatency(latencyNs)
	}
}

// RecordDeleteLatency 记录删除延迟
func (m *Metrics) RecordDeleteLatency(latencyNs int64) {
	if m.level == Disabled {
		return
	}
	atomic.AddUint64(&m.deleteLatencySum, uint64(latencyNs))
	atomic.AddUint64(&m.deleteCount, 1)
}

// UpdateEntryCount 更新条目数量
func (m *Metrics) UpdateEntryCount(count int64) {
	if m.level == Disabled {
		return
	}
	atomic.StoreInt64(&m.entryCount, count)
}

// UpdateMemoryUsage 更新内存使用量
func (m *Metrics) UpdateMemoryUsage(bytes int64) {
	if m.level == Disabled {
		return
	}
	atomic.StoreInt64(&m.memoryUsage, bytes)
}

// UpdateShardUsage 更新分片使用情况
func (m *Metrics) UpdateShardUsage(shardID int, count int64) {
	if m.level == Disabled || !m.enableShardMetrics {
		return
	}

	if shardID >= 0 && shardID < len(m.shardUsage) {
		atomic.StoreInt64(&m.shardUsage[shardID], count)

		if m.shardMetrics != nil && shardID < len(m.shardMetrics) {
			atomic.StoreInt64(&m.shardMetrics[shardID].ItemCount, count)
		}
	}
}

// UpdateShardMemoryUsage 更新分片内存使用情况
func (m *Metrics) UpdateShardMemoryUsage(shardID int, bytes int64) {
	if m.level == Disabled || !m.enableShardMetrics {
		return
	}

	if m.shardMetrics != nil && shardID >= 0 && shardID < len(m.shardMetrics) {
		atomic.StoreInt64(&m.shardMetrics[shardID].MemoryUsage, bytes)
	}
}

// RecordConflict 记录分片冲突
func (m *Metrics) RecordConflict(shardID int) {
	if m.level == Disabled || !m.enableShardMetrics {
		return
	}

	if m.shardMetrics != nil && shardID >= 0 && shardID < len(m.shardMetrics) {
		atomic.AddUint64(&m.shardMetrics[shardID].Conflicts, 1)
	}
}

// Reset 重置所有指标
func (m *Metrics) Reset() {
	if m.level == Disabled {
		return
	}

	atomic.StoreUint64(&m.hits, 0)
	atomic.StoreUint64(&m.misses, 0)
	atomic.StoreUint64(&m.hitRatio, 0)

	atomic.StoreUint64(&m.sets, 0)
	atomic.StoreUint64(&m.updates, 0)
	atomic.StoreUint64(&m.overwrites, 0)
	atomic.StoreUint64(&m.rejects, 0)

	atomic.StoreUint64(&m.evictions, 0)
	atomic.StoreUint64(&m.expired, 0)
	atomic.StoreUint64(&m.manuallyDeleted, 0)

	atomic.StoreUint64(&m.getLatencySum, 0)
	atomic.StoreUint64(&m.getCount, 0)
	atomic.StoreUint64(&m.setLatencySum, 0)
	atomic.StoreUint64(&m.setCount, 0)
	atomic.StoreUint64(&m.deleteLatencySum, 0)
	atomic.StoreUint64(&m.deleteCount, 0)

	atomic.StoreInt64(&m.lastUpdated, time.Now().UnixNano())

	// 重置分片指标
	if m.enableShardMetrics && m.shardMetrics != nil {
		for i := range m.shardMetrics {
			sm := m.shardMetrics[i]
			atomic.StoreInt64(&sm.ItemCount, 0)
			atomic.StoreInt64(&sm.MemoryUsage, 0)
			atomic.StoreUint64(&sm.Hits, 0)
			atomic.StoreUint64(&sm.Misses, 0)
			atomic.StoreUint64(&sm.Evictions, 0)
			atomic.StoreUint64(&sm.Conflicts, 0)
		}

		for i := range m.shardUsage {
			atomic.StoreInt64(&m.shardUsage[i], 0)
		}
	}

	// 重置直方图
	if m.latencyHistogram != nil {
		m.latencyHistogram.Reset()
	}
}

// GetLevel 获取当前指标采集级别
func (m *Metrics) GetLevel() Level {
	return m.level
}

// SetLevel 设置指标采集级别
func (m *Metrics) SetLevel(level Level) {
	m.level = level
}

// EnableShardMetrics 启用分片级指标
func (m *Metrics) EnableShardMetrics(shardCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enableShardMetrics = true

	if m.shardMetrics == nil || len(m.shardMetrics) != shardCount {
		m.shardMetrics = make([]*ShardMetrics, shardCount)
		m.shardUsage = make([]int64, shardCount)
		for i := 0; i < shardCount; i++ {
			m.shardMetrics[i] = &ShardMetrics{ShardID: i}
		}
	}
}

// DisableShardMetrics 禁用分片级指标
func (m *Metrics) DisableShardMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.enableShardMetrics = false
}

// EnableLatencyHistogram 启用延迟直方图
func (m *Metrics) EnableLatencyHistogram(buckets int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.latencyHistogram == nil || m.latencyHistogram.GetBucketCount() != buckets {
		m.latencyHistogram = NewHistogram(buckets)
	}
}

// DisableLatencyHistogram 禁用延迟直方图
func (m *Metrics) DisableLatencyHistogram() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latencyHistogram = nil
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() *Snapshot {
	if m.level == Disabled {
		return nil
	}

	hits := atomic.LoadUint64(&m.hits)
	misses := atomic.LoadUint64(&m.misses)
	hitRatio := float64(0)
	if hits+misses > 0 {
		hitRatio = float64(hits) / float64(hits+misses)
	}

	getLatencyAvg := int64(0)
	if atomic.LoadUint64(&m.getCount) > 0 {
		getLatencyAvg = int64(atomic.LoadUint64(&m.getLatencySum) / atomic.LoadUint64(&m.getCount))
	}

	setLatencyAvg := int64(0)
	if atomic.LoadUint64(&m.setCount) > 0 {
		setLatencyAvg = int64(atomic.LoadUint64(&m.setLatencySum) / atomic.LoadUint64(&m.setCount))
	}

	deleteLatencyAvg := int64(0)
	if atomic.LoadUint64(&m.deleteCount) > 0 {
		deleteLatencyAvg = int64(atomic.LoadUint64(&m.deleteLatencySum) / atomic.LoadUint64(&m.deleteCount))
	}

	snapshot := &Snapshot{
		Timestamp: time.Now().UnixNano(),

		Hits:     hits,
		Misses:   misses,
		HitRatio: hitRatio,

		Sets:       atomic.LoadUint64(&m.sets),
		Updates:    atomic.LoadUint64(&m.updates),
		Overwrites: atomic.LoadUint64(&m.overwrites),
		Rejects:    atomic.LoadUint64(&m.rejects),

		Evictions:       atomic.LoadUint64(&m.evictions),
		Expired:         atomic.LoadUint64(&m.expired),
		ManuallyDeleted: atomic.LoadUint64(&m.manuallyDeleted),

		GetLatencyAvg:    getLatencyAvg,
		SetLatencyAvg:    setLatencyAvg,
		DeleteLatencyAvg: deleteLatencyAvg,

		EntryCount:  atomic.LoadInt64(&m.entryCount),
		MemoryUsage: atomic.LoadInt64(&m.memoryUsage),
	}

	// 添加分片指标
	if m.enableShardMetrics && m.shardMetrics != nil {
		snapshot.ShardMetrics = make([]ShardMetricsSnapshot, len(m.shardMetrics))
		for i, sm := range m.shardMetrics {
			snapshot.ShardMetrics[i] = ShardMetricsSnapshot{
				ShardID:     sm.ShardID,
				ItemCount:   atomic.LoadInt64(&sm.ItemCount),
				MemoryUsage: atomic.LoadInt64(&sm.MemoryUsage),
				Hits:        atomic.LoadUint64(&sm.Hits),
				Misses:      atomic.LoadUint64(&sm.Misses),
				Evictions:   atomic.LoadUint64(&sm.Evictions),
				Conflicts:   atomic.LoadUint64(&sm.Conflicts),
			}
		}
	}

	// 添加直方图数据
	if m.level == Detailed && m.latencyHistogram != nil {
		snapshot.LatencyHistogram = m.latencyHistogram.GetSnapshot()
	}

	return snapshot
}

// Snapshot 指标快照
type Snapshot struct {
	Timestamp int64 `json:"timestamp"`

	// 缓存命中率相关指标
	Hits     uint64  `json:"hits"`
	Misses   uint64  `json:"misses"`
	HitRatio float64 `json:"hit_ratio"`

	// 写入行为相关指标
	Sets       uint64 `json:"sets"`
	Updates    uint64 `json:"updates"`
	Overwrites uint64 `json:"overwrites"`
	Rejects    uint64 `json:"rejects"`

	// 淘汰行为相关指标
	Evictions       uint64 `json:"evictions"`
	Expired         uint64 `json:"expired"`
	ManuallyDeleted uint64 `json:"manually_deleted"`

	// 性能指标
	GetLatencyAvg    int64 `json:"get_latency_avg_ns"`
	SetLatencyAvg    int64 `json:"set_latency_avg_ns"`
	DeleteLatencyAvg int64 `json:"delete_latency_avg_ns"`

	// 容量压力相关指标
	EntryCount  int64 `json:"entry_count"`
	MemoryUsage int64 `json:"memory_usage_bytes"`

	// 分片指标
	ShardMetrics []ShardMetricsSnapshot `json:"shard_metrics,omitempty"`

	// 延迟直方图
	LatencyHistogram *HistogramSnapshot `json:"latency_histogram,omitempty"`
}

// ShardMetricsSnapshot 分片指标快照
type ShardMetricsSnapshot struct {
	ShardID     int    `json:"shard_id"`
	ItemCount   int64  `json:"item_count"`
	MemoryUsage int64  `json:"memory_usage_bytes"`
	Hits        uint64 `json:"hits"`
	Misses      uint64 `json:"misses"`
	Evictions   uint64 `json:"evictions"`
	Conflicts   uint64 `json:"conflicts"`
}

// String 返回指标快照的JSON字符串表示
func (s *Snapshot) String() string {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// updateHitRatio 更新命中率
func (m *Metrics) updateHitRatio() {
	hits := atomic.LoadUint64(&m.hits)
	misses := atomic.LoadUint64(&m.misses)
	total := hits + misses

	if total > 0 {
		// 保留4位小数
		ratio := uint64(float64(hits) / float64(total) * 10000)
		atomic.StoreUint64(&m.hitRatio, ratio)
	} else {
		atomic.StoreUint64(&m.hitRatio, 0)
	}
}
