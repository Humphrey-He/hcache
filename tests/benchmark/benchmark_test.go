package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/yourusername/hcache/pkg/cache"
)

// TestMain 是测试的入口点
func TestMain(m *testing.M) {
	// 运行测试
	code := m.Run()
	os.Exit(code)
}

// 测试数据大小
var valueSizes = []int{
	128,   // 128B - 小数据
	1024,  // 1KB - 中等数据
	10240, // 10KB - 大数据
}

// 测试命中率场景
var hitRatioScenarios = []struct {
	name     string
	hitRatio float64
}{
	{"HighHitRatio", 0.9}, // 高命中率 (90%)
	{"MedHitRatio", 0.5},  // 中等命中率 (50%)
	{"LowHitRatio", 0.1},  // 低命中率 (10%)
}

// 初始化随机数生成器
func init() {
	rand.Seed(time.Now().UnixNano())
}

// 生成指定大小的随机字节数组
func generateRandomBytes(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// 创建测试缓存实例
func createTestCache(b *testing.B, maxEntries int) cache.ICache {
	// 使用模拟缓存实现
	return cache.NewMockCache("benchmark-cache", maxEntries, "lru")
}

// BenchmarkSet 测试设置缓存项的性能
func BenchmarkSet(b *testing.B) {
	// 禁用 GC 以减少噪音
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	ctx := context.Background()

	for _, size := range valueSizes {
		b.Run(fmt.Sprintf("ValueSize=%d", size), func(b *testing.B) {
			cacheInstance := createTestCache(b, b.N+1000)
			defer cacheInstance.Close()

			// 预生成测试数据
			testData := make([][]byte, b.N)
			for i := 0; i < b.N; i++ {
				testData[i] = generateRandomBytes(size)
			}

			b.ResetTimer()
			b.SetBytes(int64(size))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key:%d", i)
				err := cacheInstance.Set(ctx, key, testData[i], time.Hour)
				if err != nil {
					b.Fatalf("Set 操作失败: %v", err)
				}
			}
		})
	}
}

// BenchmarkGet 测试获取缓存项的性能（不同命中率场景）
func BenchmarkGet(b *testing.B) {
	// 禁用 GC 以减少噪音
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	ctx := context.Background()

	for _, size := range valueSizes {
		for _, scenario := range hitRatioScenarios {
			b.Run(fmt.Sprintf("ValueSize=%d/%s", size, scenario.name), func(b *testing.B) {
				cacheInstance := createTestCache(b, 100000)
				defer cacheInstance.Close()

				// 预填充缓存（根据目标命中率）
				preloadCount := int(float64(b.N) * scenario.hitRatio)
				for i := 0; i < preloadCount; i++ {
					key := fmt.Sprintf("key:%d", i)
					data := generateRandomBytes(size)
					err := cacheInstance.Set(ctx, key, data, time.Hour)
					if err != nil {
						b.Fatalf("预填充缓存失败: %v", err)
					}
				}

				b.ResetTimer()
				b.SetBytes(int64(size))
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					key := fmt.Sprintf("key:%d", i)
					_, _, err := cacheInstance.Get(ctx, key)
					if err != nil {
						b.Fatalf("Get 操作失败: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkMixed 测试混合操作的性能（读写混合）
func BenchmarkMixed(b *testing.B) {
	// 禁用 GC 以减少噪音
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	ctx := context.Background()

	// 不同的读写比例
	readWriteRatios := []struct {
		name      string
		readRatio float64
	}{
		{"Read90Write10", 0.9}, // 90% 读, 10% 写
		{"Read50Write50", 0.5}, // 50% 读, 50% 写
		{"Read10Write90", 0.1}, // 10% 读, 90% 写
	}

	for _, size := range valueSizes {
		for _, rwRatio := range readWriteRatios {
			b.Run(fmt.Sprintf("ValueSize=%d/%s", size, rwRatio.name), func(b *testing.B) {
				cacheInstance := createTestCache(b, 100000)
				defer cacheInstance.Close()

				// 预填充一些数据
				preloadCount := b.N / 2
				if preloadCount > 10000 {
					preloadCount = 10000 // 限制预填充数量
				}

				for i := 0; i < preloadCount; i++ {
					key := fmt.Sprintf("key:%d", i)
					data := generateRandomBytes(size)
					err := cacheInstance.Set(ctx, key, data, time.Hour)
					if err != nil {
						b.Fatalf("预填充缓存失败: %v", err)
					}
				}

				// 预生成测试数据
				testData := make([][]byte, b.N)
				for i := 0; i < b.N; i++ {
					testData[i] = generateRandomBytes(size)
				}

				b.ResetTimer()
				b.SetBytes(int64(size))
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					key := fmt.Sprintf("key:%d", i%preloadCount)

					// 根据读写比例决定操作类型
					if rand.Float64() < rwRatio.readRatio {
						// 执行读操作
						_, _, err := cacheInstance.Get(ctx, key)
						if err != nil {
							b.Fatalf("Get 操作失败: %v", err)
						}
					} else {
						// 执行写操作
						err := cacheInstance.Set(ctx, key, testData[i], time.Hour)
						if err != nil {
							b.Fatalf("Set 操作失败: %v", err)
						}
					}
				}
			})
		}
	}
}

// BenchmarkDelete 测试删除缓存项的性能
func BenchmarkDelete(b *testing.B) {
	// 禁用 GC 以减少噪音
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	ctx := context.Background()

	b.Run("Delete", func(b *testing.B) {
		cacheInstance := createTestCache(b, b.N+1000)
		defer cacheInstance.Close()

		// 预填充缓存
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key:%d", i)
			data := generateRandomBytes(128) // 使用小数据以加快预填充
			err := cacheInstance.Set(ctx, key, data, time.Hour)
			if err != nil {
				b.Fatalf("预填充缓存失败: %v", err)
			}
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key:%d", i)
			_, err := cacheInstance.Delete(ctx, key)
			if err != nil {
				b.Fatalf("Delete 操作失败: %v", err)
			}
		}
	})
}

// BenchmarkEvictionPolicy 测试不同淘汰策略的性能
func BenchmarkEvictionPolicy(b *testing.B) {
	// 禁用 GC 以减少噪音
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(runtime.NumCPU()))

	ctx := context.Background()

	policies := []string{"lru", "lfu", "fifo", "random"}

	for _, policy := range policies {
		b.Run(fmt.Sprintf("Policy=%s", policy), func(b *testing.B) {
			// 创建一个较小的缓存以触发淘汰
			cacheInstance := cache.NewMockCache("benchmark-cache", 1000, policy)
			defer cacheInstance.Close()

			b.ResetTimer()
			b.ReportAllocs()

			// 插入比缓存容量更多的数据以触发淘汰
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key:%d", i)
				data := generateRandomBytes(128)
				err := cacheInstance.Set(ctx, key, data, time.Hour)
				if err != nil {
					b.Fatalf("Set 操作失败: %v", err)
				}

				// 随机读取一些数据以影响 LRU/LFU 策略
				if i%10 == 0 {
					readKey := fmt.Sprintf("key:%d", rand.Intn(i+1))
					_, _, _ = cacheInstance.Get(ctx, readKey)
				}
			}
		})
	}
}

// 简单的测试函数，确保测试框架正常工作
func TestDummy(t *testing.T) {
	// 这个测试什么都不做，只是确保测试框架正常工作
}
