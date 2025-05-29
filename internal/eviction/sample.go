// Package eviction 提供缓存淘汰策略实现
package eviction

import (
	"math/rand"
	"sync"
	"time"
)

// SampledLFUPolicy 实现基于采样的LFU淘汰策略
// 通过随机采样减少淘汰决策的开销，适用于大规模缓存
type SampledLFUPolicy struct {
	*BasePolicy
	mu            sync.RWMutex
	items         map[uint64]*lfuHeapItem // 键到堆项的映射
	sampleSize    int                     // 采样大小
	samplingRatio float64                 // 采样比例
}

// NewSampledLFUPolicy 创建一个新的基于采样的LFU淘汰策略
func NewSampledLFUPolicy(config *Config, sampleSize int, samplingRatio float64) *SampledLFUPolicy {
	if sampleSize <= 0 {
		sampleSize = 5 // 默认采样5个
	}

	if samplingRatio <= 0 || samplingRatio > 1 {
		samplingRatio = 0.1 // 默认采样10%
	}

	return &SampledLFUPolicy{
		BasePolicy:    NewBasePolicy(config),
		items:         make(map[uint64]*lfuHeapItem),
		sampleSize:    sampleSize,
		samplingRatio: samplingRatio,
	}
}

// Add 添加一个新的条目
func (slfu *SampledLFUPolicy) Add(entry *Entry) bool {
	slfu.mu.Lock()
	defer slfu.mu.Unlock()

	// 检查是否需要淘汰
	needEvict := slfu.ShouldEvict(entry.Size)

	// 如果已存在，则更新
	if item, ok := slfu.items[entry.Key]; ok {
		slfu.UpdateSize(entry.Size-item.size, 0) // 更新大小变化
		item.value = entry.Value
		item.size = entry.Size
		item.accessTime = time.Now().UnixNano()
		item.expireAt = entry.ExpireAt
		item.frequency++
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

	// 添加到映射
	slfu.items[entry.Key] = item

	// 更新缓存大小和条目数
	slfu.UpdateSize(entry.Size, 1)

	return needEvict
}

// Get 获取一个条目，并更新其访问状态
func (slfu *SampledLFUPolicy) Get(key uint64) *Entry {
	slfu.mu.Lock()
	defer slfu.mu.Unlock()

	if item, ok := slfu.items[key]; ok {
		// 检查是否过期
		if item.IsExpired() {
			slfu.removeItem(item)
			return nil
		}

		// 更新访问时间和频率
		item.accessTime = time.Now().UnixNano()
		item.frequency++

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
func (slfu *SampledLFUPolicy) Remove(key uint64) *Entry {
	slfu.mu.Lock()
	defer slfu.mu.Unlock()

	if item, ok := slfu.items[key]; ok {
		slfu.removeItem(item)

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
func (slfu *SampledLFUPolicy) Evict(size int64) []*Entry {
	slfu.mu.Lock()
	defer slfu.mu.Unlock()

	var evicted []*Entry
	var evictedSize int64

	// 先淘汰过期条目
	var expiredItems []*lfuHeapItem
	for _, item := range slfu.items {
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
		slfu.removeItem(item)

		if evictedSize >= size {
			break
		}
	}

	// 如果淘汰过期条目后仍需要淘汰更多，则按采样LFU策略淘汰
	for evictedSize < size && len(slfu.items) > 0 {
		// 使用采样方法选择要淘汰的条目
		item := slfu.sampleAndFindMin()
		if item == nil {
			break
		}

		evicted = append(evicted, &Entry{
			Key:        item.key,
			Value:      item.value,
			Size:       item.size,
			AccessTime: item.accessTime,
			Frequency:  item.frequency,
			ExpireAt:   item.expireAt,
		})

		evictedSize += item.size
		slfu.removeItem(item)
	}

	// 更新缓存大小和条目数
	slfu.UpdateSize(-evictedSize, -len(evicted))

	return evicted
}

// Clear 清空缓存
func (slfu *SampledLFUPolicy) Clear() {
	slfu.mu.Lock()
	defer slfu.mu.Unlock()

	slfu.items = make(map[uint64]*lfuHeapItem)
	slfu.UpdateSize(-slfu.Size(), -slfu.Len())
}

// removeItem 移除一个项
func (slfu *SampledLFUPolicy) removeItem(item *lfuHeapItem) {
	delete(slfu.items, item.key)
	slfu.UpdateSize(-item.size, -1)
}

// sampleAndFindMin 采样并找到频率最低的条目
func (slfu *SampledLFUPolicy) sampleAndFindMin() *lfuHeapItem {
	if len(slfu.items) == 0 {
		return nil
	}

	// 确定采样数量
	sampleCount := int(float64(len(slfu.items)) * slfu.samplingRatio)
	if sampleCount < slfu.sampleSize {
		sampleCount = slfu.sampleSize
	}
	if sampleCount > len(slfu.items) {
		sampleCount = len(slfu.items)
	}

	// 随机选择键进行采样
	var samples []*lfuHeapItem
	keys := make([]uint64, 0, len(slfu.items))
	for k := range slfu.items {
		keys = append(keys, k)
	}

	// Fisher-Yates洗牌算法
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < sampleCount; i++ {
		j := r.Intn(len(keys)-i) + i
		keys[i], keys[j] = keys[j], keys[i]
		samples = append(samples, slfu.items[keys[i]])
	}

	// 找到频率最低的条目
	var minItem *lfuHeapItem
	minFreq := uint32(^uint32(0)) // 最大uint32值

	for _, item := range samples {
		if item.frequency < minFreq {
			minFreq = item.frequency
			minItem = item
		} else if item.frequency == minFreq && item.accessTime < minItem.accessTime {
			// 如果频率相同，则选择最早访问的
			minItem = item
		}
	}

	return minItem
}

// TinyLFUSampler 实现TinyLFU的采样器
// 用于在缓存满时决定是否接受新条目
type TinyLFUSampler struct {
	sketch        *FrequencySketch // 频率统计
	windowSize    int              // 窗口大小
	mainSize      int              // 主缓存大小
	samplingRatio float64          // 采样比例
	mu            sync.RWMutex
}

// NewTinyLFUSampler 创建一个新的TinyLFU采样器
func NewTinyLFUSampler(windowSize, mainSize int, samplingRatio float64) *TinyLFUSampler {
	if samplingRatio <= 0 || samplingRatio > 1 {
		samplingRatio = 0.01 // 默认采样1%
	}

	return &TinyLFUSampler{
		sketch:        NewFrequencySketch(4, 16),
		windowSize:    windowSize,
		mainSize:      mainSize,
		samplingRatio: samplingRatio,
	}
}

// Record 记录一个键的访问
func (s *TinyLFUSampler) Record(key uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sketch.Increment(key)
}

// Sample 采样并决定是否接受新条目
// candidate 是候选条目，victim 是可能被淘汰的条目
// 返回true表示接受候选条目，false表示拒绝
func (s *TinyLFUSampler) Sample(candidate, victim uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 估计频率
	candidateFreq := s.sketch.Estimate(candidate)
	victimFreq := s.sketch.Estimate(victim)

	// 如果候选条目的频率大于受害者，则接受
	return candidateFreq > victimFreq
}

// Reset 重置采样器
func (s *TinyLFUSampler) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sketch.Reset()
}

// RandomSampler 实现随机采样器
// 用于从大量条目中随机选择一部分进行操作
type RandomSampler struct {
	sampleSize int     // 采样大小
	ratio      float64 // 采样比例
	rnd        *rand.Rand
	mu         sync.Mutex
}

// NewRandomSampler 创建一个新的随机采样器
func NewRandomSampler(sampleSize int, ratio float64) *RandomSampler {
	if sampleSize <= 0 {
		sampleSize = 5
	}

	if ratio <= 0 || ratio > 1 {
		ratio = 0.1
	}

	return &RandomSampler{
		sampleSize: sampleSize,
		ratio:      ratio,
		rnd:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SampleKeys 从键集合中采样
func (s *RandomSampler) SampleKeys(keys []uint64) []uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确定采样数量
	sampleCount := int(float64(len(keys)) * s.ratio)
	if sampleCount < s.sampleSize {
		sampleCount = s.sampleSize
	}
	if sampleCount > len(keys) {
		sampleCount = len(keys)
	}

	// 创建结果切片
	result := make([]uint64, sampleCount)

	// Fisher-Yates洗牌算法
	keysCopy := make([]uint64, len(keys))
	copy(keysCopy, keys)

	for i := 0; i < sampleCount; i++ {
		j := s.rnd.Intn(len(keysCopy)-i) + i
		keysCopy[i], keysCopy[j] = keysCopy[j], keysCopy[i]
		result[i] = keysCopy[i]
	}

	return result
}

// SampleItems 从项集合中采样
func (s *RandomSampler) SampleItems(items map[uint64]*lfuHeapItem) []*lfuHeapItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确定采样数量
	sampleCount := int(float64(len(items)) * s.ratio)
	if sampleCount < s.sampleSize {
		sampleCount = s.sampleSize
	}
	if sampleCount > len(items) {
		sampleCount = len(items)
	}

	// 创建结果切片
	result := make([]*lfuHeapItem, 0, sampleCount)

	// 获取所有键
	keys := make([]uint64, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}

	// Fisher-Yates洗牌算法
	for i := 0; i < sampleCount; i++ {
		j := s.rnd.Intn(len(keys)-i) + i
		keys[i], keys[j] = keys[j], keys[i]
		result = append(result, items[keys[i]])
	}

	return result
}

// SetSeed 设置随机数生成器的种子
func (s *RandomSampler) SetSeed(seed int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rnd = rand.New(rand.NewSource(seed))
}
