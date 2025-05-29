// Package admission 提供缓存准入控制机制，用于判断数据是否应该被缓存
// 主要基于Count-Min Sketch算法实现频率估计，以提高缓存命中率
package admission

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// 默认哈希函数数量
	defaultDepth = 4
	// 默认每个哈希函数的计数器宽度
	defaultWidth = 1024
	// 默认重置周期（以访问次数计）
	defaultResetAfter uint64 = 100000
)

// CountMinSketch 实现了一个并发安全的Count-Min Sketch算法
// 用于高效估计元素的频率，作为缓存准入控制的基础
type CountMinSketch struct {
	depth      int          // 哈希函数数量
	width      int          // 每个哈希函数的计数器宽度
	matrix     [][]uint64   // 计数矩阵，使用uint64以支持原子操作
	seeds      []uint64     // 哈希函数种子
	count      uint64       // 当前已处理的元素计数
	resetAfter uint64       // 重置周期
	mutex      sync.RWMutex // 用于重置操作的锁
	doorkeeper *BloomFilter // 门卫过滤器，减少对稀有项目的计数
}

// Config 定义Count-Min Sketch的配置选项
type Config struct {
	// 哈希函数数量，影响精度
	Depth int
	// 每个哈希函数的计数器宽度，影响冲突率
	Width int
	// 重置周期，以访问次数计
	ResetAfter uint64
	// 是否启用BloomFilter作为前置过滤
	EnableDoorkeeper bool
}

// NewCountMinSketch 创建一个新的Count-Min Sketch实例
func NewCountMinSketch(config *Config) *CountMinSketch {
	depth := defaultDepth
	width := defaultWidth
	resetAfter := defaultResetAfter

	if config != nil {
		if config.Depth > 0 {
			depth = config.Depth
		}
		if config.Width > 0 {
			width = config.Width
		}
		if config.ResetAfter > 0 {
			resetAfter = config.ResetAfter
		}
	}

	// 初始化计数矩阵
	matrix := make([][]uint64, depth)
	for i := 0; i < depth; i++ {
		matrix[i] = make([]uint64, width)
	}

	// 初始化哈希种子
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	seeds := make([]uint64, depth)
	for i := 0; i < depth; i++ {
		seeds[i] = uint64(r.Int63())
	}

	var doorkeeper *BloomFilter
	if config != nil && config.EnableDoorkeeper {
		doorkeeper = NewBloomFilter(uint64(width*8), 3) // 简单的BloomFilter配置
	}

	return &CountMinSketch{
		depth:      depth,
		width:      width,
		matrix:     matrix,
		seeds:      seeds,
		resetAfter: resetAfter,
		doorkeeper: doorkeeper,
	}
}

// Increment 增加一个键的计数
func (cms *CountMinSketch) Increment(key uint64) {
	// 如果启用了doorkeeper且是首次见到该key，则先记录
	if cms.doorkeeper != nil && !cms.doorkeeper.Contains(key) {
		cms.doorkeeper.Add(key)
		return
	}

	// 对每个哈希函数
	for i := 0; i < cms.depth; i++ {
		// 计算哈希值
		hash := cms.hash(key, cms.seeds[i]) % uint64(cms.width)
		// 原子递增对应计数器
		atomic.AddUint64(&cms.matrix[i][hash], 1)
	}

	// 原子递增总计数
	newCount := atomic.AddUint64(&cms.count, 1)

	// 检查是否需要重置
	if newCount >= cms.resetAfter {
		cms.tryReset()
	}
}

// Estimate 估计一个键的频率
func (cms *CountMinSketch) Estimate(key uint64) uint64 {
	// 如果启用了doorkeeper且不包含该key，则返回0
	if cms.doorkeeper != nil && !cms.doorkeeper.Contains(key) {
		return 0
	}

	cms.mutex.RLock()
	defer cms.mutex.RUnlock()

	min := uint64(math.MaxUint64)
	// 取所有哈希函数计数的最小值作为估计值
	for i := 0; i < cms.depth; i++ {
		hash := cms.hash(key, cms.seeds[i]) % uint64(cms.width)
		count := atomic.LoadUint64(&cms.matrix[i][hash])
		if count < min {
			min = count
		}
	}

	return min
}

// Reset 重置所有计数器
func (cms *CountMinSketch) Reset() {
	cms.mutex.Lock()
	defer cms.mutex.Unlock()

	// 重置所有计数器
	for i := 0; i < cms.depth; i++ {
		for j := 0; j < cms.width; j++ {
			cms.matrix[i][j] = 0
		}
	}

	atomic.StoreUint64(&cms.count, 0)

	// 重置doorkeeper
	if cms.doorkeeper != nil {
		cms.doorkeeper.Reset()
	}
}

// tryReset 尝试重置计数器，使用CAS操作避免并发问题
func (cms *CountMinSketch) tryReset() {
	// 尝试获取锁进行重置
	if cms.mutex.TryLock() {
		defer cms.mutex.Unlock()

		// 再次检查计数，避免重复重置
		if atomic.LoadUint64(&cms.count) >= cms.resetAfter {
			// 对所有计数器进行衰减而不是完全重置
			// 这样可以保留热点数据的相对频率
			for i := 0; i < cms.depth; i++ {
				for j := 0; j < cms.width; j++ {
					// 衰减为原来的一半
					atomic.StoreUint64(&cms.matrix[i][j], cms.matrix[i][j]/2)
				}
			}

			atomic.StoreUint64(&cms.count, cms.resetAfter/2)
		}
	}
}

// hash 计算哈希值
func (cms *CountMinSketch) hash(key, seed uint64) uint64 {
	// FNV-1a哈希算法
	h := seed
	h ^= (key & 0xff)
	h *= 0x100000001b3
	h ^= (key >> 8) & 0xff
	h *= 0x100000001b3
	h ^= (key >> 16) & 0xff
	h *= 0x100000001b3
	h ^= (key >> 24) & 0xff
	h *= 0x100000001b3
	h ^= (key >> 32) & 0xff
	h *= 0x100000001b3
	h ^= (key >> 40) & 0xff
	h *= 0x100000001b3
	h ^= (key >> 48) & 0xff
	h *= 0x100000001b3
	h ^= (key >> 56) & 0xff
	h *= 0x100000001b3

	return h
}

// BloomFilter 实现简单的布隆过滤器，用作门卫减少对稀有项目的计数
type BloomFilter struct {
	bits   []uint64
	size   uint64
	hashes int
	mutex  sync.RWMutex
}

// NewBloomFilter 创建一个新的布隆过滤器
func NewBloomFilter(size uint64, hashes int) *BloomFilter {
	return &BloomFilter{
		bits:   make([]uint64, (size+63)/64), // 向上取整到64位边界
		size:   size,
		hashes: hashes,
	}
}

// Add 添加一个元素到布隆过滤器
func (bf *BloomFilter) Add(key uint64) {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	for i := 0; i < bf.hashes; i++ {
		hash := bf.hash(key, uint64(i)) % bf.size
		bf.bits[hash/64] |= 1 << (hash % 64)
	}
}

// Contains 检查布隆过滤器是否可能包含某元素
func (bf *BloomFilter) Contains(key uint64) bool {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	for i := 0; i < bf.hashes; i++ {
		hash := bf.hash(key, uint64(i)) % bf.size
		if (bf.bits[hash/64] & (1 << (hash % 64))) == 0 {
			return false
		}
	}

	return true
}

// Reset 重置布隆过滤器
func (bf *BloomFilter) Reset() {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	for i := range bf.bits {
		bf.bits[i] = 0
	}
}

// hash 为布隆过滤器计算哈希值
func (bf *BloomFilter) hash(key, seed uint64) uint64 {
	// MurmurHash简化版
	h := seed ^ (key * 0xc6a4a7935bd1e995)
	h ^= h >> 47
	h *= 0xc6a4a7935bd1e995
	return h
}
