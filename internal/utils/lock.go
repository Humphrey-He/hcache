// Package utils 提供HCache内部使用的通用工具函数
package utils

import (
	"context"
	"sync"
	"time"
)

// TryMutex 是一个支持尝试获取锁的互斥锁
// 在高并发场景下，可以避免长时间阻塞
type TryMutex struct {
	mu sync.Mutex
}

// Lock 获取锁，会阻塞直到获取成功
func (tm *TryMutex) Lock() {
	tm.mu.Lock()
}

// Unlock 释放锁
func (tm *TryMutex) Unlock() {
	tm.mu.Unlock()
}

// TryLock 尝试获取锁，如果获取成功返回true，否则返回false
// 此方法不会阻塞
func (tm *TryMutex) TryLock() bool {
	return tm.TryLockTimeout(0)
}

// TryLockTimeout 尝试在指定的超时时间内获取锁
// 如果获取成功返回true，否则返回false
func (tm *TryMutex) TryLockTimeout(timeout time.Duration) bool {
	if timeout <= 0 {
		// 尝试获取锁，不等待
		ch := make(chan bool, 1)
		go func() {
			tm.mu.Lock()
			ch <- true
		}()

		select {
		case <-ch:
			return true
		default:
			return false
		}
	}

	// 带超时的锁获取
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan bool, 1)
	go func() {
		tm.mu.Lock()
		ch <- true
	}()

	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

// TryRWMutex 是一个支持尝试获取锁的读写锁
type TryRWMutex struct {
	mu sync.RWMutex
}

// Lock 获取写锁，会阻塞直到获取成功
func (trw *TryRWMutex) Lock() {
	trw.mu.Lock()
}

// Unlock 释放写锁
func (trw *TryRWMutex) Unlock() {
	trw.mu.Unlock()
}

// RLock 获取读锁，会阻塞直到获取成功
func (trw *TryRWMutex) RLock() {
	trw.mu.RLock()
}

// RUnlock 释放读锁
func (trw *TryRWMutex) RUnlock() {
	trw.mu.RUnlock()
}

// TryLock 尝试获取写锁，如果获取成功返回true，否则返回false
func (trw *TryRWMutex) TryLock() bool {
	return trw.TryLockTimeout(0)
}

// TryLockTimeout 尝试在指定的超时时间内获取写锁
func (trw *TryRWMutex) TryLockTimeout(timeout time.Duration) bool {
	if timeout <= 0 {
		ch := make(chan bool, 1)
		go func() {
			trw.mu.Lock()
			ch <- true
		}()

		select {
		case <-ch:
			return true
		default:
			return false
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan bool, 1)
	go func() {
		trw.mu.Lock()
		ch <- true
	}()

	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

// TryRLock 尝试获取读锁，如果获取成功返回true，否则返回false
func (trw *TryRWMutex) TryRLock() bool {
	return trw.TryRLockTimeout(0)
}

// TryRLockTimeout 尝试在指定的超时时间内获取读锁
func (trw *TryRWMutex) TryRLockTimeout(timeout time.Duration) bool {
	if timeout <= 0 {
		ch := make(chan bool, 1)
		go func() {
			trw.mu.RLock()
			ch <- true
		}()

		select {
		case <-ch:
			return true
		default:
			return false
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan bool, 1)
	go func() {
		trw.mu.RLock()
		ch <- true
	}()

	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

// NamedLock 是一个命名锁，用于对特定资源进行锁定
// 适用于需要对大量不同资源进行锁定的场景
type NamedLock struct {
	locks     map[string]*sync.Mutex
	lockCount map[string]int
	mu        sync.Mutex
}

// NewNamedLock 创建一个新的命名锁
func NewNamedLock() *NamedLock {
	return &NamedLock{
		locks:     make(map[string]*sync.Mutex),
		lockCount: make(map[string]int),
	}
}

// Lock 锁定指定名称的资源
func (nl *NamedLock) Lock(name string) {
	// 获取或创建锁
	mu := nl.getMutex(name)
	mu.Lock()
}

// Unlock 解锁指定名称的资源
func (nl *NamedLock) Unlock(name string) {
	nl.mu.Lock()
	mu, ok := nl.locks[name]
	if !ok {
		nl.mu.Unlock()
		return
	}

	// 减少锁计数
	nl.lockCount[name]--
	if nl.lockCount[name] <= 0 {
		delete(nl.locks, name)
		delete(nl.lockCount, name)
	}
	nl.mu.Unlock()

	mu.Unlock()
}

// TryLock 尝试锁定指定名称的资源
func (nl *NamedLock) TryLock(name string) bool {
	// 获取或创建锁
	mu := nl.getMutex(name)

	// 尝试获取锁
	ch := make(chan bool, 1)
	go func() {
		mu.Lock()
		ch <- true
	}()

	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// getMutex 获取或创建指定名称的锁
func (nl *NamedLock) getMutex(name string) *sync.Mutex {
	nl.mu.Lock()
	defer nl.mu.Unlock()

	mu, ok := nl.locks[name]
	if !ok {
		mu = &sync.Mutex{}
		nl.locks[name] = mu
		nl.lockCount[name] = 0
	}
	nl.lockCount[name]++
	return mu
}

// ShardedLock 是一个分片锁，用于减少锁竞争
// 适用于需要对大量资源进行锁定的场景
type ShardedLock struct {
	locks     []*sync.Mutex
	shardMask uint64
}

// NewShardedLock 创建一个新的分片锁
// shardCount 必须是2的幂次方
func NewShardedLock(shardCount int) *ShardedLock {
	if shardCount <= 0 {
		shardCount = 16 // 默认16个分片
	}

	// 确保分片数量是2的幂次方
	shardCount = NextPowerOfTwo(shardCount)
	locks := make([]*sync.Mutex, shardCount)
	for i := 0; i < shardCount; i++ {
		locks[i] = &sync.Mutex{}
	}

	return &ShardedLock{
		locks:     locks,
		shardMask: uint64(shardCount - 1),
	}
}

// Lock 锁定指定键对应的分片
func (sl *ShardedLock) Lock(key uint64) {
	shardIndex := key & sl.shardMask
	sl.locks[shardIndex].Lock()
}

// Unlock 解锁指定键对应的分片
func (sl *ShardedLock) Unlock(key uint64) {
	shardIndex := key & sl.shardMask
	sl.locks[shardIndex].Unlock()
}

// LockAll 锁定所有分片
// 注意：为避免死锁，必须按顺序锁定
func (sl *ShardedLock) LockAll() {
	for _, mu := range sl.locks {
		mu.Lock()
	}
}

// UnlockAll 解锁所有分片
// 注意：为避免死锁，必须按与锁定相反的顺序解锁
func (sl *ShardedLock) UnlockAll() {
	for i := len(sl.locks) - 1; i >= 0; i-- {
		sl.locks[i].Unlock()
	}
}

// ShardCount 返回分片数量
func (sl *ShardedLock) ShardCount() int {
	return len(sl.locks)
}

// DeadlockDetector 是一个简单的死锁检测器
// 用于在开发和测试环境中检测潜在的死锁
type DeadlockDetector struct {
	timeout time.Duration
	enabled bool
	mu      sync.Mutex
	locks   map[string]time.Time
}

// NewDeadlockDetector 创建一个新的死锁检测器
// timeout 是锁定超时时间，超过此时间将被视为潜在死锁
func NewDeadlockDetector(timeout time.Duration) *DeadlockDetector {
	if timeout <= 0 {
		timeout = 5 * time.Second // 默认5秒
	}

	dd := &DeadlockDetector{
		timeout: timeout,
		enabled: true,
		locks:   make(map[string]time.Time),
	}

	// 启动检测协程
	go dd.detector()

	return dd
}

// Enable 启用死锁检测
func (dd *DeadlockDetector) Enable() {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.enabled = true
}

// Disable 禁用死锁检测
func (dd *DeadlockDetector) Disable() {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.enabled = false
}

// BeforeLock 在获取锁之前调用
func (dd *DeadlockDetector) BeforeLock(name string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if !dd.enabled {
		return
	}

	dd.locks[name] = time.Now()
}

// AfterLock 在获取锁之后调用
func (dd *DeadlockDetector) AfterLock(name string) {
	dd.mu.Lock()
	defer dd.mu.Unlock()

	if !dd.enabled {
		return
	}

	delete(dd.locks, name)
}

// detector 检测潜在的死锁
func (dd *DeadlockDetector) detector() {
	ticker := time.NewTicker(dd.timeout / 2)
	defer ticker.Stop()

	for range ticker.C {
		dd.mu.Lock()
		if !dd.enabled {
			dd.mu.Unlock()
			continue
		}

		now := time.Now()
		for name, lockTime := range dd.locks {
			if now.Sub(lockTime) > dd.timeout {
				// 检测到潜在死锁
				// 在实际应用中，可以记录日志或触发警报
				delete(dd.locks, name)
			}
		}
		dd.mu.Unlock()
	}
}
