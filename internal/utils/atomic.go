// Package utils 提供HCache内部使用的通用工具函数
package utils

import (
	"sync/atomic"
)

// AtomicInt64 是int64类型的原子包装器
// 提供更易用的原子操作接口
type AtomicInt64 struct {
	value int64
}

// NewAtomicInt64 创建一个新的AtomicInt64
func NewAtomicInt64(initialValue int64) *AtomicInt64 {
	return &AtomicInt64{value: initialValue}
}

// Get 原子地获取值
func (a *AtomicInt64) Get() int64 {
	return atomic.LoadInt64(&a.value)
}

// Set 原子地设置值
func (a *AtomicInt64) Set(newValue int64) {
	atomic.StoreInt64(&a.value, newValue)
}

// Add 原子地增加值并返回新值
func (a *AtomicInt64) Add(delta int64) int64 {
	return atomic.AddInt64(&a.value, delta)
}

// Swap 原子地交换值并返回旧值
func (a *AtomicInt64) Swap(newValue int64) int64 {
	return atomic.SwapInt64(&a.value, newValue)
}

// CompareAndSwap 原子地比较并交换值
// 如果当前值等于oldValue，则设置为newValue并返回true
// 否则返回false
func (a *AtomicInt64) CompareAndSwap(oldValue, newValue int64) bool {
	return atomic.CompareAndSwapInt64(&a.value, oldValue, newValue)
}

// AtomicUint64 是uint64类型的原子包装器
type AtomicUint64 struct {
	value uint64
}

// NewAtomicUint64 创建一个新的AtomicUint64
func NewAtomicUint64(initialValue uint64) *AtomicUint64 {
	return &AtomicUint64{value: initialValue}
}

// Get 原子地获取值
func (a *AtomicUint64) Get() uint64 {
	return atomic.LoadUint64(&a.value)
}

// Set 原子地设置值
func (a *AtomicUint64) Set(newValue uint64) {
	atomic.StoreUint64(&a.value, newValue)
}

// Add 原子地增加值并返回新值
func (a *AtomicUint64) Add(delta uint64) uint64 {
	return atomic.AddUint64(&a.value, delta)
}

// Swap 原子地交换值并返回旧值
func (a *AtomicUint64) Swap(newValue uint64) uint64 {
	return atomic.SwapUint64(&a.value, newValue)
}

// CompareAndSwap 原子地比较并交换值
func (a *AtomicUint64) CompareAndSwap(oldValue, newValue uint64) bool {
	return atomic.CompareAndSwapUint64(&a.value, oldValue, newValue)
}

// AtomicBool 是bool类型的原子包装器
// 内部使用int32表示，0表示false，1表示true
type AtomicBool struct {
	value int32
}

// NewAtomicBool 创建一个新的AtomicBool
func NewAtomicBool(initialValue bool) *AtomicBool {
	var value int32
	if initialValue {
		value = 1
	}
	return &AtomicBool{value: value}
}

// Get 原子地获取值
func (a *AtomicBool) Get() bool {
	return atomic.LoadInt32(&a.value) != 0
}

// Set 原子地设置值
func (a *AtomicBool) Set(newValue bool) {
	var value int32
	if newValue {
		value = 1
	}
	atomic.StoreInt32(&a.value, value)
}

// Swap 原子地交换值并返回旧值
func (a *AtomicBool) Swap(newValue bool) bool {
	var value int32
	if newValue {
		value = 1
	}
	return atomic.SwapInt32(&a.value, value) != 0
}

// CompareAndSwap 原子地比较并交换值
func (a *AtomicBool) CompareAndSwap(oldValue, newValue bool) bool {
	var oldInt32, newInt32 int32
	if oldValue {
		oldInt32 = 1
	}
	if newValue {
		newInt32 = 1
	}
	return atomic.CompareAndSwapInt32(&a.value, oldInt32, newInt32)
}

// TrySet 尝试将值从false设置为true
// 如果成功返回true，否则返回false
func (a *AtomicBool) TrySet() bool {
	return atomic.CompareAndSwapInt32(&a.value, 0, 1)
}

// TryUnset 尝试将值从true设置为false
// 如果成功返回true，否则返回false
func (a *AtomicBool) TryUnset() bool {
	return atomic.CompareAndSwapInt32(&a.value, 1, 0)
}

// AtomicPointer 是指针类型的原子包装器
// 使用unsafe.Pointer实现
type AtomicPointer struct {
	value atomic.Value
}

// NewAtomicPointer 创建一个新的AtomicPointer
func NewAtomicPointer(initialValue interface{}) *AtomicPointer {
	a := &AtomicPointer{}
	if initialValue != nil {
		a.value.Store(initialValue)
	}
	return a
}

// Get 原子地获取值
func (a *AtomicPointer) Get() interface{} {
	return a.value.Load()
}

// Set 原子地设置值
func (a *AtomicPointer) Set(newValue interface{}) {
	a.value.Store(newValue)
}

// Swap 原子地交换值并返回旧值
func (a *AtomicPointer) Swap(newValue interface{}) interface{} {
	return a.value.Swap(newValue)
}

// CompareAndSwap 原子地比较并交换值
func (a *AtomicPointer) CompareAndSwap(oldValue, newValue interface{}) bool {
	return a.value.CompareAndSwap(oldValue, newValue)
}

// RetryUpdate 重试执行原子更新操作，直到成功
// updateFn接收当前值并返回新值
// 返回最终的新值
func RetryUpdate[T comparable](addr *atomic.Value, updateFn func(T) T) T {
	for {
		oldValue := addr.Load().(T)
		newValue := updateFn(oldValue)
		if addr.CompareAndSwap(oldValue, newValue) {
			return newValue
		}
	}
}

// RetryUpdateWithLimit 重试执行原子更新操作，直到成功或达到最大尝试次数
// updateFn接收当前值并返回新值
// 返回最终的新值和是否成功
func RetryUpdateWithLimit[T comparable](addr *atomic.Value, updateFn func(T) T, maxRetries int) (T, bool) {
	for i := 0; i < maxRetries; i++ {
		oldValue := addr.Load().(T)
		newValue := updateFn(oldValue)
		if addr.CompareAndSwap(oldValue, newValue) {
			return newValue, true
		}
	}
	return addr.Load().(T), false
}
