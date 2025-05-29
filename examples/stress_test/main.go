// Package main implements a stress testing tool for the HCache library.
// It simulates high load with configurable concurrency, read/write ratios,
// and access patterns to evaluate cache performance under pressure.
//
// Package main 实现了HCache库的压力测试工具。
// 它模拟高负载，具有可配置的并发性、读/写比率和访问模式，以评估缓存在压力下的性能。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/hcache/pkg/cache"
)

// StressTestConfig holds the configuration for the stress test.
// It defines all parameters that control the behavior and intensity
// of the cache stress test.
//
// StressTestConfig 保存压力测试的配置。
// 它定义了控制缓存压力测试行为和强度的所有参数。
type StressTestConfig struct {
	QPS            int           // Target QPS / 目标QPS
	Duration       time.Duration // Test duration / 测试持续时间
	KeySpace       int           // Number of unique keys to use / 要使用的唯一键数量
	Workers        int           // Number of concurrent workers / 并发工作线程数
	ValueSize      int           // Size of values in bytes / 值的大小（字节）
	ReadPercentage int           // Percentage of read operations (vs writes) / 读操作百分比（相对于写操作）
	ReportInterval time.Duration // Interval for reporting stats / 报告统计信息的间隔
	OutputFormat   string        // Output format (text, csv, markdown) / 输出格式（文本、CSV、Markdown）
}

// StressTestStats holds the statistics for the stress test.
// It tracks various metrics during the test execution to provide
// insights into cache performance.
//
// StressTestStats 保存压力测试的统计信息。
// 它在测试执行期间跟踪各种指标，以提供对缓存性能的洞察。
type StressTestStats struct {
	TotalRequests      int64      // Total number of requests / 请求总数
	SuccessfulRequests int64      // Number of successful requests / 成功请求数
	FailedRequests     int64      // Number of failed requests / 失败请求数
	ReadRequests       int64      // Number of read requests / 读请求数
	WriteRequests      int64      // Number of write requests / 写请求数
	CacheHits          int64      // Number of cache hits / 缓存命中数
	CacheMisses        int64      // Number of cache misses / 缓存未命中数
	TotalLatency       int64      // Total latency in nanoseconds / 总延迟（纳秒）
	MaxLatency         int64      // Maximum latency in nanoseconds / 最大延迟（纳秒）
	P95Latency         int64      // 95th percentile latency in nanoseconds / 95百分位延迟（纳秒）
	StartTime          time.Time  // Start time of the test / 测试开始时间
	Latencies          []int64    // All latencies in nanoseconds, for percentile calculation / 所有延迟（纳秒），用于百分位计算
	mu                 sync.Mutex // Mutex for thread-safe access to latencies / 用于对延迟进行线程安全访问的互斥锁
}

// main is the entry point for the stress test application.
// It parses command line flags, initializes the cache, and runs the stress test.
//
// main 是压力测试应用程序的入口点。
// 它解析命令行标志，初始化缓存，并运行压力测试。
func main() {
	// Parse command line flags
	// 解析命令行标志
	qps := flag.Int("qps", 1000, "Target QPS")
	duration := flag.Duration("duration", 30*time.Second, "Test duration")
	keySpace := flag.Int("keys", 10000, "Number of unique keys to use")
	workers := flag.Int("workers", 10, "Number of concurrent workers")
	valueSize := flag.Int("value-size", 1024, "Size of values in bytes")
	readPercentage := flag.Int("read-pct", 80, "Percentage of read operations (vs writes)")
	reportInterval := flag.Duration("report-interval", 1*time.Second, "Interval for reporting stats")
	outputFormat := flag.String("output", "text", "Output format (text, csv, markdown)")
	cacheSize := flag.Int("cache-size", 100000, "Maximum number of cache entries")
	ttl := flag.Duration("ttl", 5*time.Minute, "Default TTL for cache entries")
	shards := flag.Int("shards", 16, "Number of cache shards")
	flag.Parse()

	// Validate inputs to ensure they are within acceptable ranges
	// 验证输入以确保它们在可接受的范围内
	if *qps <= 0 {
		log.Fatal("QPS must be positive")
	}
	if *keySpace <= 0 {
		log.Fatal("Key space must be positive")
	}
	if *workers <= 0 {
		log.Fatal("Number of workers must be positive")
	}
	if *readPercentage < 0 || *readPercentage > 100 {
		log.Fatal("Read percentage must be between 0 and 100")
	}

	// Create stress test config from command line parameters
	// 从命令行参数创建压力测试配置
	config := StressTestConfig{
		QPS:            *qps,
		Duration:       *duration,
		KeySpace:       *keySpace,
		Workers:        *workers,
		ValueSize:      *valueSize,
		ReadPercentage: *readPercentage,
		ReportInterval: *reportInterval,
		OutputFormat:   *outputFormat,
	}

	// Initialize cache with the specified options
	// 使用指定的选项初始化缓存
	cacheInstance, err := cache.NewWithOptions("stress-test-cache",
		cache.WithMaxEntryCount(*cacheSize),
		cache.WithTTL(*ttl),
		cache.WithShards(*shards),
		cache.WithMetricsEnabled(true),
	)
	if err != nil {
		log.Fatalf("Failed to create cache: %v", err)
	}
	defer cacheInstance.Close()

	// Display test configuration
	// 显示测试配置
	fmt.Println("Starting stress test with the following configuration:")
	fmt.Printf("  QPS: %d\n", config.QPS)
	fmt.Printf("  Duration: %s\n", config.Duration)
	fmt.Printf("  Key space: %d\n", config.KeySpace)
	fmt.Printf("  Workers: %d\n", config.Workers)
	fmt.Printf("  Value size: %d bytes\n", config.ValueSize)
	fmt.Printf("  Read percentage: %d%%\n", config.ReadPercentage)
	fmt.Printf("  Cache size: %d entries\n", *cacheSize)
	fmt.Printf("  TTL: %s\n", *ttl)
	fmt.Printf("  Shards: %d\n", *shards)
	fmt.Println()

	// Run the stress test and collect statistics
	// 运行压力测试并收集统计信息
	stats := runStressTest(cacheInstance, config)

	// Print final results in the specified format
	// 以指定格式打印最终结果
	printResults(stats, config)
}

// runStressTest runs the stress test with the given configuration.
// It spawns worker goroutines, manages rate limiting, and collects statistics.
//
// Parameters:
//   - cacheInstance: The cache instance to test
//   - config: The stress test configuration
//
// Returns:
//   - *StressTestStats: The collected statistics from the test
//
// runStressTest 使用给定的配置运行压力测试。
// 它生成工作线程，管理速率限制，并收集统计信息。
//
// 参数:
//   - cacheInstance: 要测试的缓存实例
//   - config: 压力测试配置
//
// 返回:
//   - *StressTestStats: 测试收集的统计信息
func runStressTest(cacheInstance cache.ICache, config StressTestConfig) *StressTestStats {
	// Initialize statistics collection
	// 初始化统计信息收集
	stats := &StressTestStats{
		StartTime: time.Now(),
		Latencies: make([]int64, 0, config.QPS*int(config.Duration.Seconds())),
	}

	// Create context with cancellation for coordinating test shutdown
	// 创建带有取消功能的上下文，用于协调测试关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C to gracefully terminate the test
	// 处理Ctrl+C以优雅地终止测试
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, stopping test...")
		cancel()
	}()

	// Create a channel for rate limiting to achieve target QPS
	// 创建一个用于速率限制的通道，以实现目标QPS
	ticker := time.NewTicker(time.Second / time.Duration(config.QPS))
	defer ticker.Stop()

	// Create a channel for periodic reporting of statistics
	// 创建一个用于定期报告统计信息的通道
	reportTicker := time.NewTicker(config.ReportInterval)
	defer reportTicker.Stop()

	// Create a wait group for synchronizing worker goroutines
	// 创建一个等待组，用于同步工作线程
	var wg sync.WaitGroup

	// Start worker goroutines
	// 启动工作线程
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go worker(ctx, i, cacheInstance, config, stats, ticker.C, &wg)
	}

	// Start reporter goroutine to periodically output statistics
	// 启动报告线程，定期输出统计信息
	go reporter(ctx, cacheInstance, stats, reportTicker.C, config)

	// Wait for test duration or cancellation
	// 等待测试持续时间或取消
	select {
	case <-time.After(config.Duration):
		cancel()
	case <-ctx.Done():
		// Context was cancelled externally
		// 上下文被外部取消
	}

	// Wait for all workers to finish
	// 等待所有工作线程完成
	wg.Wait()

	return stats
}

// worker performs read and write operations on the cache.
// Each worker represents a client accessing the cache concurrently.
//
// Parameters:
//   - ctx: Context for cancellation
//   - id: Worker identifier
//   - cacheInstance: The cache instance to test
//   - config: The stress test configuration
//   - stats: Statistics collection
//   - rateLimiter: Channel for controlling operation rate
//   - wg: WaitGroup for worker synchronization
//
// worker 在缓存上执行读写操作。
// 每个工作线程代表一个并发访问缓存的客户端。
//
// 参数:
//   - ctx: 用于取消的上下文
//   - id: 工作线程标识符
//   - cacheInstance: 要测试的缓存实例
//   - config: 压力测试配置
//   - stats: 统计信息收集
//   - rateLimiter: 用于控制操作速率的通道
//   - wg: 用于工作线程同步的WaitGroup
func worker(
	ctx context.Context,
	id int,
	cacheInstance cache.ICache,
	config StressTestConfig,
	stats *StressTestStats,
	rateLimiter <-chan time.Time,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	// Create a random value of the specified size
	// 创建指定大小的随机值
	value := make([]byte, config.ValueSize)
	rand.Read(value)

	for {
		select {
		case <-ctx.Done():
			return
		case <-rateLimiter:
			// Decide whether to do a read or write operation
			// 决定是执行读操作还是写操作
			isRead := rand.Intn(100) < config.ReadPercentage

			// Choose a random key
			// 选择一个随机键
			key := fmt.Sprintf("key:%d", rand.Intn(config.KeySpace))

			// Start timing
			// 开始计时
			startTime := time.Now()

			// Perform the operation
			// 执行操作
			var err error
			if isRead {
				// Read operation
				// 读操作
				atomic.AddInt64(&stats.ReadRequests, 1)
				val, exists, readErr := cacheInstance.Get(ctx, key)
				err = readErr
				if err == nil {
					if exists {
						atomic.AddInt64(&stats.CacheHits, 1)
						_ = val // Use the value to prevent compiler optimization
					} else {
						atomic.AddInt64(&stats.CacheMisses, 1)
					}
				}
			} else {
				// Write operation
				// 写操作
				atomic.AddInt64(&stats.WriteRequests, 1)
				err = cacheInstance.Set(ctx, key, value, 5*time.Minute)
			}

			// Record latency
			// 记录延迟
			latency := time.Since(startTime).Nanoseconds()
			recordLatency(stats, latency)

			// Update request counters
			// 更新请求计数器
			atomic.AddInt64(&stats.TotalRequests, 1)
			if err == nil {
				atomic.AddInt64(&stats.SuccessfulRequests, 1)
			} else {
				atomic.AddInt64(&stats.FailedRequests, 1)
			}
		}
	}
}

// recordLatency records a latency measurement and updates statistics.
// It atomically updates the total and maximum latency, and adds the
// measurement to the latencies slice for percentile calculations.
//
// Parameters:
//   - stats: The statistics object to update
//   - latency: The latency measurement in nanoseconds
//
// recordLatency 记录延迟测量并更新统计信息。
// 它以原子方式更新总延迟和最大延迟，并将测量值添加到延迟切片中以进行百分位计算。
//
// 参数:
//   - stats: 要更新的统计对象
//   - latency: 延迟测量值（纳秒）
func recordLatency(stats *StressTestStats, latency int64) {
	atomic.AddInt64(&stats.TotalLatency, latency)

	// Update max latency (using atomic compare-and-swap)
	// 更新最大延迟（使用原子比较和交换）
	for {
		current := atomic.LoadInt64(&stats.MaxLatency)
		if latency <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&stats.MaxLatency, current, latency) {
			break
		}
	}

	// Add to latencies slice for percentile calculation
	// 添加到延迟切片中以进行百分位计算
	stats.mu.Lock()
	stats.Latencies = append(stats.Latencies, latency)
	stats.mu.Unlock()
}

// reporter periodically reports statistics during the stress test.
// It calculates and displays metrics like requests per second, success rate,
// cache hit rate, and latency statistics at regular intervals.
//
// Parameters:
//   - ctx: Context for cancellation
//   - cacheInstance: The cache instance being tested
//   - stats: The statistics object to report from
//   - reportTicker: Channel that triggers when a report should be generated
//   - config: The stress test configuration
//
// reporter 在压力测试期间定期报告统计信息。
// 它计算并定期显示指标，如每秒请求数、成功率、缓存命中率和延迟统计信息。
//
// 参数:
//   - ctx: 用于取消的上下文
//   - cacheInstance: 正在测试的缓存实例
//   - stats: 用于报告的统计对象
//   - reportTicker: 在应生成报告时触发的通道
//   - config: 压力测试配置
func reporter(
	ctx context.Context,
	cacheInstance cache.ICache,
	stats *StressTestStats,
	reportTicker <-chan time.Time,
	config StressTestConfig,
) {
	var lastRequests int64
	var lastTime time.Time = time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-reportTicker:
			now := time.Now()
			currentRequests := atomic.LoadInt64(&stats.TotalRequests)
			elapsed := now.Sub(lastTime)
			requestsInPeriod := currentRequests - lastRequests
			rps := float64(requestsInPeriod) / elapsed.Seconds()

			// Calculate stats for this reporting period
			// 计算此报告期间的统计信息
			totalRequests := atomic.LoadInt64(&stats.TotalRequests)
			successRate := float64(atomic.LoadInt64(&stats.SuccessfulRequests)) / float64(totalRequests) * 100
			readRate := float64(atomic.LoadInt64(&stats.ReadRequests)) / float64(totalRequests) * 100
			writeRate := float64(atomic.LoadInt64(&stats.WriteRequests)) / float64(totalRequests) * 100

			hitCount := atomic.LoadInt64(&stats.CacheHits)
			missCount := atomic.LoadInt64(&stats.CacheMisses)
			hitRate := float64(0)
			if hitCount+missCount > 0 {
				hitRate = float64(hitCount) / float64(hitCount+missCount) * 100
			}

			avgLatency := float64(0)
			if totalRequests > 0 {
				avgLatency = float64(atomic.LoadInt64(&stats.TotalLatency)) / float64(totalRequests) / float64(time.Millisecond)
			}
			maxLatency := float64(atomic.LoadInt64(&stats.MaxLatency)) / float64(time.Millisecond)

			// Calculate p95 latency
			// 计算P95延迟
			p95Latency := calculateP95Latency(stats)

			// Get cache stats from the cache instance
			// 从缓存实例获取缓存统计信息
			cacheStats, _ := cacheInstance.Stats(ctx)

			// Print report to console
			// 将报告打印到控制台
			fmt.Printf("[%s] Requests: %d (%.2f req/s), Success: %.2f%%, Reads: %.2f%%, Writes: %.2f%%, Hit rate: %.2f%%\n",
				now.Format("15:04:05"),
				totalRequests,
				rps,
				successRate,
				readRate,
				writeRate,
				hitRate,
			)
			fmt.Printf("         Latency: avg=%.2f ms, p95=%.2f ms, max=%.2f ms\n",
				avgLatency,
				p95Latency,
				maxLatency,
			)
			fmt.Printf("         Cache: entries=%d, size=%d bytes\n",
				cacheStats.EntryCount,
				cacheStats.Size,
			)
			fmt.Println()

			// Update last values for next calculation
			// 更新上次值以供下次计算
			lastRequests = currentRequests
			lastTime = now
		}
	}
}

// calculateP95Latency calculates the 95th percentile latency.
// It creates a sorted copy of the latency measurements and
// returns the value at the 95th percentile position.
//
// Parameters:
//   - stats: The statistics object containing latency measurements
//
// Returns:
//   - float64: The 95th percentile latency in milliseconds
//
// calculateP95Latency 计算第95百分位延迟。
// 它创建延迟测量的排序副本，并返回第95百分位位置的值。
//
// 参数:
//   - stats: 包含延迟测量的统计对象
//
// 返回:
//   - float64: 第95百分位延迟（毫秒）
func calculateP95Latency(stats *StressTestStats) float64 {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	if len(stats.Latencies) == 0 {
		return 0
	}

	// Sort latencies (we'll use a simple selection sort since we don't want to modify the original slice)
	// 排序延迟（我们将使用简单的选择排序，因为我们不想修改原始切片）
	n := len(stats.Latencies)
	if n == 0 {
		return 0
	}

	// Create a copy to sort
	// 创建副本进行排序
	sorted := make([]int64, n)
	copy(sorted, stats.Latencies)

	// Simple insertion sort
	// 简单的插入排序
	for i := 1; i < n; i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	// Calculate p95 index
	// 计算P95索引
	p95Index := int(float64(n) * 0.95)
	if p95Index >= n {
		p95Index = n - 1
	}

	return float64(sorted[p95Index]) / float64(time.Millisecond)
}

// printResults prints the final test results in the specified format.
// It calculates overall statistics and outputs them according to the
// configured format (text, CSV, or Markdown).
//
// Parameters:
//   - stats: The statistics object containing test results
//   - config: The stress test configuration
//
// printResults 以指定格式打印最终测试结果。
// 它计算总体统计信息，并根据配置的格式（文本、CSV或Markdown）输出它们。
//
// 参数:
//   - stats: 包含测试结果的统计对象
//   - config: 压力测试配置
func printResults(stats *StressTestStats, config StressTestConfig) {
	duration := time.Since(stats.StartTime)
	totalRequests := atomic.LoadInt64(&stats.TotalRequests)
	successfulRequests := atomic.LoadInt64(&stats.SuccessfulRequests)
	failedRequests := atomic.LoadInt64(&stats.FailedRequests)
	readRequests := atomic.LoadInt64(&stats.ReadRequests)
	writeRequests := atomic.LoadInt64(&stats.WriteRequests)
	cacheHits := atomic.LoadInt64(&stats.CacheHits)
	cacheMisses := atomic.LoadInt64(&stats.CacheMisses)

	// Calculate final metrics
	// 计算最终指标
	rps := float64(totalRequests) / duration.Seconds()
	successRate := float64(successfulRequests) / float64(totalRequests) * 100
	readRate := float64(readRequests) / float64(totalRequests) * 100
	writeRate := float64(writeRequests) / float64(totalRequests) * 100

	hitRate := float64(0)
	if cacheHits+cacheMisses > 0 {
		hitRate = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	}

	avgLatency := float64(0)
	if totalRequests > 0 {
		avgLatency = float64(atomic.LoadInt64(&stats.TotalLatency)) / float64(totalRequests) / float64(time.Millisecond)
	}
	maxLatency := float64(atomic.LoadInt64(&stats.MaxLatency)) / float64(time.Millisecond)
	p95Latency := calculateP95Latency(stats)

	// Output results in the specified format
	// 以指定格式输出结果
	switch config.OutputFormat {
	case "csv":
		outputCSV(stats, config, duration, rps, successRate, hitRate, avgLatency, p95Latency, maxLatency)
	case "markdown":
		outputMarkdown(stats, config, duration, rps, successRate, hitRate, avgLatency, p95Latency, maxLatency)
	default:
		// Text output (default)
		// 文本输出（默认）
		fmt.Println("\nTest Results:")
		fmt.Printf("Duration: %s\n", duration.Round(time.Millisecond))
		fmt.Printf("Total Requests: %d\n", totalRequests)
		fmt.Printf("Successful Requests: %d (%.2f%%)\n", successfulRequests, successRate)
		fmt.Printf("Failed Requests: %d (%.2f%%)\n", failedRequests, 100-successRate)
		fmt.Printf("Read Requests: %d (%.2f%%)\n", readRequests, readRate)
		fmt.Printf("Write Requests: %d (%.2f%%)\n", writeRequests, writeRate)
		fmt.Printf("Cache Hits: %d\n", cacheHits)
		fmt.Printf("Cache Misses: %d\n", cacheMisses)
		fmt.Printf("Cache Hit Rate: %.2f%%\n", hitRate)
		fmt.Printf("Requests Per Second: %.2f\n", rps)
		fmt.Printf("Average Latency: %.2f ms\n", avgLatency)
		fmt.Printf("P95 Latency: %.2f ms\n", p95Latency)
		fmt.Printf("Max Latency: %.2f ms\n", maxLatency)
	}
}

// outputCSV outputs the test results in CSV format.
// This is useful for importing results into spreadsheets or
// other data analysis tools.
//
// Parameters:
//   - stats: The statistics object containing test results
//   - config: The stress test configuration
//   - duration: The total test duration
//   - rps: Requests per second
//   - successRate: Percentage of successful requests
//   - hitRate: Cache hit rate percentage
//   - avgLatency: Average latency in milliseconds
//   - p95Latency: 95th percentile latency in milliseconds
//   - maxLatency: Maximum latency in milliseconds
//
// outputCSV 以CSV格式输出测试结果。
// 这对于将结果导入电子表格或其他数据分析工具很有用。
//
// 参数:
//   - stats: 包含测试结果的统计对象
//   - config: 压力测试配置
//   - duration: 总测试持续时间
//   - rps: 每秒请求数
//   - successRate: 成功请求的百分比
//   - hitRate: 缓存命中率百分比
//   - avgLatency: 平均延迟（毫秒）
//   - p95Latency: 第95百分位延迟（毫秒）
//   - maxLatency: 最大延迟（毫秒）
func outputCSV(stats *StressTestStats, config StressTestConfig, duration time.Duration, rps, successRate, hitRate, avgLatency, p95Latency, maxLatency float64) {
	// Print CSV header if this is the first run
	// 如果这是第一次运行，则打印CSV标题
	fmt.Println("timestamp,qps,workers,keyspace,valuesize,readpct,duration,totalreqs,successrate,hitrate,avglat,p95lat,maxlat")

	// Print CSV row
	// 打印CSV行
	fmt.Printf("%d,%d,%d,%d,%d,%d,%d,%d,%.2f,%.2f,%.2f,%.2f,%.2f\n",
		time.Now().Unix(),
		config.QPS,
		config.Workers,
		config.KeySpace,
		config.ValueSize,
		config.ReadPercentage,
		int(duration.Seconds()),
		stats.TotalRequests,
		successRate,
		hitRate,
		avgLatency,
		p95Latency,
		maxLatency,
	)
}

// outputMarkdown outputs the test results in Markdown format.
// This is useful for including results in documentation or
// sharing in platforms that support Markdown.
//
// Parameters:
//   - stats: The statistics object containing test results
//   - config: The stress test configuration
//   - duration: The total test duration
//   - rps: Requests per second
//   - successRate: Percentage of successful requests
//   - hitRate: Cache hit rate percentage
//   - avgLatency: Average latency in milliseconds
//   - p95Latency: 95th percentile latency in milliseconds
//   - maxLatency: Maximum latency in milliseconds
//
// outputMarkdown 以Markdown格式输出测试结果。
// 这对于在文档中包含结果或在支持Markdown的平台上共享很有用。
//
// 参数:
//   - stats: 包含测试结果的统计对象
//   - config: 压力测试配置
//   - duration: 总测试持续时间
//   - rps: 每秒请求数
//   - successRate: 成功请求的百分比
//   - hitRate: 缓存命中率百分比
//   - avgLatency: 平均延迟（毫秒）
//   - p95Latency: 第95百分位延迟（毫秒）
//   - maxLatency: 最大延迟（毫秒）
func outputMarkdown(stats *StressTestStats, config StressTestConfig, duration time.Duration, rps, successRate, hitRate, avgLatency, p95Latency, maxLatency float64) {
	fmt.Println("# HCache Stress Test Results")
	fmt.Println()
	fmt.Println("## Test Configuration")
	fmt.Println()
	fmt.Println("| Parameter | Value |")
	fmt.Println("|-----------|-------|")
	fmt.Printf("| QPS | %d |\n", config.QPS)
	fmt.Printf("| Workers | %d |\n", config.Workers)
	fmt.Printf("| Key Space | %d |\n", config.KeySpace)
	fmt.Printf("| Value Size | %d bytes |\n", config.ValueSize)
	fmt.Printf("| Read Percentage | %d%% |\n", config.ReadPercentage)
	fmt.Printf("| Duration | %s |\n", duration.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("## Results")
	fmt.Println()
	fmt.Println("| Metric | Value |")
	fmt.Println("|--------|-------|")
	fmt.Printf("| Total Requests | %d |\n", stats.TotalRequests)
	fmt.Printf("| Requests Per Second | %.2f |\n", rps)
	fmt.Printf("| Success Rate | %.2f%% |\n", successRate)
	fmt.Printf("| Cache Hit Rate | %.2f%% |\n", hitRate)
	fmt.Printf("| Average Latency | %.2f ms |\n", avgLatency)
	fmt.Printf("| P95 Latency | %.2f ms |\n", p95Latency)
	fmt.Printf("| Max Latency | %.2f ms |\n", maxLatency)
	fmt.Printf("| Read/Write Ratio | %d/%d |\n", config.ReadPercentage, 100-config.ReadPercentage)
}
