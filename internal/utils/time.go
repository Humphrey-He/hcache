// Package utils 提供HCache内部使用的通用工具函数
package utils

import (
	"sync"
	"sync/atomic"
	"time"
)

var (
	// 缓存的当前时间（纳秒）
	cachedTimeNano int64
	// 缓存的当前时间（秒）
	cachedTimeSec int64
	// 是否已启动时间更新协程
	timeUpdaterStarted int32
)

// 初始化时间缓存
func init() {
	// 设置初始值
	now := time.Now()
	atomic.StoreInt64(&cachedTimeNano, now.UnixNano())
	atomic.StoreInt64(&cachedTimeSec, now.Unix())

	// 启动更新协程
	startTimeUpdater()
}

// startTimeUpdater 启动时间更新协程
// 使用CAS确保只启动一次
func startTimeUpdater() {
	if atomic.CompareAndSwapInt32(&timeUpdaterStarted, 0, 1) {
		go func() {
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				now := time.Now()
				atomic.StoreInt64(&cachedTimeNano, now.UnixNano())
				atomic.StoreInt64(&cachedTimeSec, now.Unix())
			}
		}()
	}
}

// NowNano 返回当前时间（纳秒）
// 使用缓存的时间，避免频繁调用time.Now()
func NowNano() int64 {
	return atomic.LoadInt64(&cachedTimeNano)
}

// NowSec 返回当前时间（秒）
// 使用缓存的时间，避免频繁调用time.Now()
func NowSec() int64 {
	return atomic.LoadInt64(&cachedTimeSec)
}

// AlignedTicker 创建一个与时钟对齐的定时器
// 例如，如果duration为1分钟，则定时器将在每分钟的整点触发
func AlignedTicker(duration time.Duration) *time.Ticker {
	now := time.Now()
	nextTick := now.Truncate(duration).Add(duration)
	delay := nextTick.Sub(now)

	// 先等待到下一个整点
	time.Sleep(delay)

	// 然后创建定时器
	return time.NewTicker(duration)
}

// ExpiryBucket 计算过期时间所在的桶
// 用于将过期时间分组，减少过期检查的频率
// bucketSize是桶的大小（秒）
func ExpiryBucket(expireAt int64, bucketSize int64) int64 {
	if expireAt <= 0 {
		return 0
	}
	return expireAt / bucketSize
}

// SleepUntil 睡眠到指定的时间
func SleepUntil(targetTime time.Time) {
	now := time.Now()
	if now.Before(targetTime) {
		time.Sleep(targetTime.Sub(now))
	}
}

// TimeWindow 表示一个滑动时间窗口
type TimeWindow struct {
	duration time.Duration // 窗口持续时间
	buckets  int           // 桶数量
	counts   []int64       // 每个桶的计数
	times    []int64       // 每个桶的时间戳
	current  int           // 当前桶索引
	total    int64         // 总计数
	mu       sync.RWMutex  // 互斥锁
}

// NewTimeWindow 创建一个新的滑动时间窗口
// duration是窗口的总持续时间
// buckets是窗口内的桶数量
func NewTimeWindow(duration time.Duration, buckets int) *TimeWindow {
	if buckets <= 0 {
		buckets = 10 // 默认10个桶
	}

	return &TimeWindow{
		duration: duration,
		buckets:  buckets,
		counts:   make([]int64, buckets),
		times:    make([]int64, buckets),
		current:  0,
		total:    0,
	}
}

// Add 增加计数
func (w *TimeWindow) Add(delta int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := NowNano()
	w.rotate(now)

	w.counts[w.current] += delta
	w.total += delta
}

// Count 获取窗口内的总计数
func (w *TimeWindow) Count() int64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	now := NowNano()
	w.rotate(now)

	return w.total
}

// Rate 获取窗口内的平均速率（每秒）
func (w *TimeWindow) Rate() float64 {
	w.mu.RLock()
	defer w.mu.RUnlock()

	now := NowNano()
	w.rotate(now)

	seconds := float64(w.duration) / float64(time.Second)
	return float64(w.total) / seconds
}

// Reset 重置窗口
func (w *TimeWindow) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i := range w.counts {
		w.counts[i] = 0
		w.times[i] = 0
	}

	w.total = 0
	w.current = 0
}

// rotate 旋转窗口，清理过期的桶
func (w *TimeWindow) rotate(now int64) {
	// 计算每个桶的持续时间
	bucketDuration := w.duration / time.Duration(w.buckets)

	// 计算最早的有效时间
	oldest := now - int64(w.duration)

	// 检查并更新过期的桶
	for i := 0; i < w.buckets; i++ {
		idx := (w.current + i + 1) % w.buckets

		// 如果桶为空或已过期，则清零
		if w.times[idx] < oldest {
			w.total -= w.counts[idx]
			w.counts[idx] = 0
			w.times[idx] = 0
		}
	}

	// 更新当前桶
	w.current = int((now / int64(bucketDuration)) % int64(w.buckets))

	// 如果当前桶是新的，则初始化
	if w.times[w.current] == 0 {
		w.times[w.current] = now
	}
}

// TimedValue 表示一个带有过期时间的值
type TimedValue[T any] struct {
	Value    T
	ExpireAt int64
}

// NewTimedValue 创建一个新的带过期时间的值
func NewTimedValue[T any](value T, ttl time.Duration) TimedValue[T] {
	return TimedValue[T]{
		Value:    value,
		ExpireAt: NowNano() + int64(ttl),
	}
}

// IsExpired 检查值是否已过期
func (tv *TimedValue[T]) IsExpired() bool {
	if tv.ExpireAt == 0 {
		return false
	}
	return NowNano() > tv.ExpireAt
}

// TTL 返回剩余的生存时间
func (tv *TimedValue[T]) TTL() time.Duration {
	if tv.ExpireAt == 0 {
		return 0
	}

	remaining := tv.ExpireAt - NowNano()
	if remaining <= 0 {
		return 0
	}

	return time.Duration(remaining)
}

// Extend 延长过期时间
func (tv *TimedValue[T]) Extend(ttl time.Duration) {
	if ttl <= 0 {
		return
	}

	if tv.ExpireAt == 0 {
		tv.ExpireAt = NowNano() + int64(ttl)
	} else {
		tv.ExpireAt += int64(ttl)
	}
}
