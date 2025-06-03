# HCache

<div align="center">

[![Go Reference](https://pkg.go.dev/badge/github.com/Humphrey-He/hcache.svg)](https://pkg.go.dev/github.com/Humphrey-He/hcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/Humphrey-He/hcache)](https://goreportcard.com/report/github.com/Humphrey-He/hcache)
[![License](https://img.shields.io/github/license/Humphrey-He/hcache)](LICENSE)
[![Build Status](https://github.com/Humphrey-He/hcache/workflows/build/badge.svg)](https://github.com/Humphrey-He/hcache/actions)
[![Coverage](https://codecov.io/gh/Humphrey-He/hcache/branch/main/graph/badge.svg)](https://codecov.io/gh/Humphrey-He/hcache)

<p>A high-performance, feature-rich in-memory cache library for Go applications</p>
</div>

## 📋 Table of Contents

- [Project Structure](#-project-structure)
- [Key Features](#-key-features)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Detailed Usage](#-detailed-usage)
  - [Basic Operations](#basic-operations)
  - [Cache Configuration](#cache-configuration)
  - [Eviction Policies](#eviction-policies)
  - [TTL Management](#ttl-management)
  - [Data Loading Strategies](#data-loading-strategies)
  - [Metrics and Monitoring](#metrics-and-monitoring)
  - [Advanced Features](#advanced-features)
- [Integration Examples](#-integration-examples)
  - [HTTP Server Integration](#http-server-integration)
  - [Concurrent Applications](#concurrent-applications)
- [Performance Benchmarks](#-performance-benchmarks)
  - [Test Environment](#test-environment)
  - [Core Operations Performance](#core-operations-performance)
  - [Eviction Policies Comparison](#eviction-policies-comparison)
  - [Concurrency Performance](#concurrency-performance)
  - [Memory Efficiency](#memory-efficiency)
- [Contributing](#-contributing)
- [License](#-license)

## 📂 Project Structure

HCache is organized with a clear separation between public APIs and internal implementations:

```
/pkg                  - Public APIs for external use
  /cache              - Main cache interfaces and implementations
    cache.go          - Core cache interface definitions
    config.go         - Configuration structures
    options.go        - Functional options for configuration
    factory.go        - Factory methods for cache creation
    mock.go           - Mock implementations for testing
  /codec              - Serialization interfaces and codecs
  /errors             - Error definitions and handling
  /loader             - Data loading interfaces and implementations

/internal             - Internal implementations (not for external use)
  /eviction           - Cache eviction policy implementations
    policy.go         - Eviction policy interfaces
    lfu.go            - Least Frequently Used implementation
    wtinyLFU.go       - Window TinyLFU implementation
    sample.go         - Sampling-based eviction implementations
  /storage            - Storage implementations
    store.go          - Core storage interfaces
    optimize.go       - Performance optimizations
  /ttl                - TTL management and expiration
  /metrics            - Performance metrics collection
  /admission          - Admission policy implementations
  /utils              - Utility functions and data structures

/examples             - Example applications
  /benchmark          - Performance benchmarking code
  /http_server        - HTTP server integration example
  /stress_test        - Stress testing utilities

/tests                - Test suites and utilities
```

This structure ensures a clean separation between the stable public API in `/pkg` and the implementation details in `/internal`, allowing for future optimizations without breaking API compatibility.

## ✨ Key Features

HCache stands out with the following key features:

- **Ultra-Low Latency**: Core operations optimized for nanosecond-level performance
  - Get/Hit: ~97 ns/op with zero memory allocations
  - Get/Miss: ~125 ns/op with minimal allocations
  - Set operations: ~440 ns/op for new entries

- **Multiple Eviction Strategies**: Choose the best algorithm for your workload
  - **LRU** (Least Recently Used): Optimized for recency-based access patterns
  - **LFU** (Least Frequently Used): Ideal for frequency-based workloads
  - **FIFO** (First In First Out): Simple and efficient for sequential access
  - **Random**: Lowest overhead option for large caches

- **Advanced Concurrency Model**: Designed for high-throughput multi-core systems
  - Sharded architecture minimizes lock contention
  - Fine-grained locking for maximum parallelism
  - Thread-safe operations with minimal synchronization overhead

- **Comprehensive Memory Management**: Prevents OOM situations
  - Configurable entry count limits
  - Byte-level memory usage tracking and limits
  - Efficient memory allocation patterns (0 B/op for read operations)

- **Sophisticated Data Loading**: Multiple strategies for handling cache misses
  - Synchronous and asynchronous loading
  - Batch loading for related keys
  - Fallback mechanisms for resilience

- **Detailed Performance Metrics**: Insights for optimization
  - Hit/miss statistics with ratio calculation
  - Latency tracking for operations
  - Eviction and expiration metrics

## 📦 Installation

```bash
go get github.com/Humphrey-He/hcache
```

## 🚀 Quick Start

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Humphrey-He/hcache/pkg/cache"
)

func main() {
	// Create a cache with default configuration
	c, err := cache.NewWithOptions("myCache",
		cache.WithMaxEntryCount(1000),
		cache.WithTTL(time.Minute*5),
	)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	ctx := context.Background()

	// Set a value
	err = c.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		panic(err)
	}

	// Get a value
	value, exists, err := c.Get(ctx, "key1")
	if err != nil {
		panic(err)
	}

	if exists {
		fmt.Printf("Value: %v\n", value)
	} else {
		fmt.Println("Key not found")
	}
}
```

## 📝 Detailed Usage

### Basic Operations

HCache provides a clean, context-aware API for all cache operations:

```go
// Create context
ctx := context.Background()

// Set with explicit TTL
cache.Set(ctx, "user:1001", userData, time.Hour)

// Set with default TTL
cache.Set(ctx, "session:abc", sessionData, 0)

// Get with type assertion
value, exists, err := cache.Get(ctx, "user:1001")
if exists {
    user := value.(UserData)
    fmt.Println(user.Name)
}

// Delete
removed, err := cache.Delete(ctx, "user:1001")

// Clear all entries
cache.Clear(ctx)

// Get stats
stats, err := cache.Stats(ctx)
fmt.Printf("Hits: %d, Misses: %d, Ratio: %.2f%%\n", 
           stats.Hits, stats.Misses, float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100)
```

### Cache Configuration

HCache uses a functional options pattern for flexible configuration:

```go
cache, err := cache.NewWithOptions("userCache",
    // Capacity settings
    cache.WithMaxEntryCount(100000),          // Maximum number of entries
    cache.WithMaxMemoryBytes(500*1024*1024),  // Memory limit (500MB)
    
    // Performance settings
    cache.WithShards(256),                    // Number of shards for concurrency
    cache.WithShardedLock(true),              // Enable fine-grained locking
    
    // Eviction settings
    cache.WithEviction("lru"),                // LRU, LFU, FIFO, or Random
    cache.WithTTL(time.Hour),                 // Default TTL
    cache.WithCleanupInterval(time.Minute*5), // How often to check for expired entries
    
    // Advanced settings
    cache.WithMetricsEnabled(true),           // Enable metrics collection
    cache.WithMetricsLevel("detailed"),       // Metrics detail level
    cache.WithAdmissionPolicy(true),          // Enable admission policy
    
    // Serialization settings
    cache.WithCompression(true),              // Enable value compression
    cache.WithCompressionThreshold(1024),     // Compress values larger than 1KB
    cache.WithCodec(myCodec),                 // Custom serialization
    
    // Data loading
    cache.WithLoader(myLoader),               // Data loader for cache misses
)
```

### Eviction Policies

HCache supports multiple eviction policies to match different access patterns:

```go
// LRU - Least Recently Used
// Best for recency-based access patterns where recently accessed items are likely to be accessed again
lruCache, _ := cache.NewWithOptions("lruCache", cache.WithEviction("lru"))

// LFU - Least Frequently Used
// Best for frequency-based access patterns with stable popularity distribution
lfuCache, _ := cache.NewWithOptions("lfuCache", cache.WithEviction("lfu"))

// FIFO - First In First Out
// Simple queue-based eviction, good for sequential access patterns
fifoCache, _ := cache.NewWithOptions("fifoCache", cache.WithEviction("fifo"))

// Random - Random Eviction
// Lowest overhead, good for large caches with uniform access patterns
randomCache, _ := cache.NewWithOptions("randomCache", cache.WithEviction("random"))
```

Each policy has different performance characteristics and memory overhead:

- **LRU**: Maintains access recency information, moderate memory overhead
- **LFU**: Tracks access frequency, higher CPU usage for counter maintenance
- **FIFO**: Simplest implementation, lowest CPU overhead
- **Random**: Lowest memory overhead, but potentially lower hit rates

### TTL Management

HCache provides flexible TTL (Time-To-Live) management:

```go
// Set default TTL for all entries
cache, _ := cache.NewWithOptions("sessionCache", 
    cache.WithTTL(time.Minute * 30))

// Set item with custom TTL
cache.Set(ctx, "short-lived", value, time.Second * 10)

// Set item with infinite TTL (no expiration)
cache.Set(ctx, "permanent", value, -1)

// Set item with default TTL
cache.Set(ctx, "default-ttl", value, 0)

// Configure background cleanup of expired items
cache, _ := cache.NewWithOptions("cleanCache",
    cache.WithTTL(time.Hour),
    cache.WithCleanupInterval(time.Minute * 5))
```

TTL is enforced in two ways:
1. **On access**: Expired items are detected and removed during Get operations
2. **Background cleanup**: A janitor goroutine periodically removes expired items

### Data Loading Strategies

HCache provides several strategies for loading data on cache misses:

```go
import "github.com/Humphrey-He/hcache/pkg/loader"

// Simple function loader
simpleLoader := loader.NewFunctionLoader(func(ctx context.Context, key string) (interface{}, error) {
    return fetchFromDatabase(key)
})

// Function loader with custom TTL
ttlLoader := loader.NewFunctionLoaderWithTTL(func(ctx context.Context, key string) (interface{}, time.Duration, error) {
    data, err := fetchFromDatabase(key)
    return data, time.Hour, err  // Custom TTL per item
})

// Fallback loader (primary with backup)
fallbackLoader := loader.NewFallbackLoader(
    primaryLoader,
    backupLoader,
)

// Cached loader (with local in-memory buffer)
cachedLoader := loader.NewCachedLoader(
    databaseLoader,
    time.Minute,  // Local buffer TTL
)

// Using a loader with cache
c, _ := cache.NewWithOptions("dbCache",
    cache.WithLoader(simpleLoader),
)

// GetOrLoad automatically uses the loader on cache miss
userData, err := c.GetOrLoad(ctx, "user:1001")
```

For batch operations, implement the BatchLoader interface:

```go
type UserBatchLoader struct{}

func (l *UserBatchLoader) LoadBatch(ctx context.Context, keys []string) (map[string]interface{}, map[string]time.Duration, error) {
    // Fetch multiple users in one database query
    users, err := fetchUsersFromDatabase(keys)
    
    // Convert to result maps
    values := make(map[string]interface{})
    ttls := make(map[string]time.Duration)
    
    for k, user := range users {
        values[k] = user
        ttls[k] = time.Hour
    }
    
    return values, ttls, err
}
```

### Metrics and Monitoring

HCache provides detailed metrics for monitoring cache performance:

```go
// Enable metrics collection
c, _ := cache.NewWithOptions("monitoredCache",
    cache.WithMetricsEnabled(true),
    cache.WithMetricsLevel("detailed"),  // "basic", "detailed", or "disabled"
)

// Get basic stats
stats, _ := c.Stats(ctx)
fmt.Printf("Entries: %d\n", stats.EntryCount)
fmt.Printf("Hit ratio: %.2f%%\n", float64(stats.Hits)/(float64(stats.Hits+stats.Misses))*100)
fmt.Printf("Memory usage: %.2f MB\n", float64(stats.Size)/(1024*1024))
fmt.Printf("Evictions: %d\n", stats.Evictions)

// With detailed metrics enabled, additional information is available
detailedStats := stats.(*cache.DetailedStats)  // Type assertion for detailed stats
fmt.Printf("Average get latency: %v\n", detailedStats.AvgGetLatency)
fmt.Printf("P99 get latency: %v\n", detailedStats.P99GetLatency)
fmt.Printf("Cache fragmentation: %.2f%%\n", detailedStats.FragmentationRatio*100)
```

HCache can also export metrics to Prometheus:

```go
import "github.com/Humphrey-He/hcache/pkg/metrics"

// Register cache metrics with Prometheus
metrics.RegisterPrometheus(c, "myapp_cache")
```

### Advanced Features

#### Custom Serialization

```go
import "github.com/Humphrey-He/hcache/pkg/codec"

// Use built-in JSON codec
jsonCodec := codec.NewJSONCodec()

// Use built-in Gob codec (faster for Go types)
gobCodec := codec.NewGobCodec()

// Use built-in Protocol Buffers codec
protobufCodec := codec.NewProtobufCodec()

// Create custom codec by implementing the Codec interface
type MyCustomCodec struct{}

func (c *MyCustomCodec) Marshal(v interface{}) ([]byte, error) {
    // Custom serialization logic
    return mySerialize(v)
}

func (c *MyCustomCodec) Unmarshal(data []byte, v interface{}) error {
    // Custom deserialization logic
    return myDeserialize(data, v)
}

// Use codec with cache
c, _ := cache.NewWithOptions("myCache",
    cache.WithCodec(&MyCustomCodec{}),
)
```

#### Admission Policy

Admission policies help prevent cache pollution by low-value items:

```go
// Enable admission policy
c, _ := cache.NewWithOptions("protectedCache",
    cache.WithAdmissionPolicy(true),  // Uses TinyLFU by default
)
```

The admission policy tracks access frequency patterns and only admits items that are likely to be accessed multiple times, protecting the cache from scan-based workloads and one-time access patterns.

#### Compression

For large values, HCache can automatically compress data:

```go
// Enable compression for values over 4KB
c, _ := cache.NewWithOptions("compressedCache",
    cache.WithCompression(true),
    cache.WithCompressionThreshold(4*1024),  // 4KB
)
```

Compression reduces memory usage at the cost of slightly increased CPU usage during Get/Set operations for large values.

## 🔌 Integration Examples

### HTTP Server Integration

HCache can be integrated with HTTP servers to cache API responses:

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
    
    "github.com/Humphrey-He/hcache/pkg/cache"
)

type Product struct {
    ID    string  `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

func main() {
    // Create cache
    productCache, _ := cache.NewWithOptions("products",
        cache.WithMaxEntryCount(1000),
        cache.WithTTL(time.Minute * 5),
    )
    
    // Product handler with caching
    http.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
        productID := r.URL.Path[len("/products/"):]
        ctx := r.Context()
        
        // Try to get from cache
        cachedProduct, exists, err := productCache.Get(ctx, productID)
        if err != nil {
            http.Error(w, "Cache error", http.StatusInternalServerError)
            return
        }
        
        var product Product
        
        if exists {
            // Cache hit
            product = cachedProduct.(Product)
        } else {
            // Cache miss - fetch from database
            product, err = fetchProductFromDB(productID)
            if err != nil {
                http.Error(w, "Database error", http.StatusInternalServerError)
                return
            }
            
            // Store in cache
            productCache.Set(ctx, productID, product, 0)  // Use default TTL
        }
        
        // Return JSON response
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(product)
    })
    
    // Cache stats endpoint
    http.HandleFunc("/cache/stats", func(w http.ResponseWriter, r *http.Request) {
        stats, _ := productCache.Stats(r.Context())
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(stats)
    })
    
    http.ListenAndServe(":8080", nil)
}

func fetchProductFromDB(id string) (Product, error) {
    // Simulate database access
    time.Sleep(100 * time.Millisecond)
    return Product{ID: id, Name: "Product " + id, Price: 99.99}, nil
}
```

See the [examples/http_server](examples/http_server) directory for a complete example.

### Concurrent Applications

HCache is designed for high-concurrency scenarios:

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "github.com/Humphrey-He/hcache/pkg/cache"
)

func main() {
    // Create cache with many shards for high concurrency
    c, _ := cache.NewWithOptions("concurrentCache",
        cache.WithMaxEntryCount(100000),
        cache.WithShards(256),  // 256 shards for minimal lock contention
    )
    
    ctx := context.Background()
    var wg sync.WaitGroup
    
    // Simulate 100 concurrent goroutines
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            // Each goroutine performs 1000 operations
            for j := 0; j < 1000; j++ {
                key := fmt.Sprintf("key:%d:%d", id, j)
                
                // 80% reads, 20% writes
                if j%5 == 0 {
                    c.Set(ctx, key, fmt.Sprintf("value:%d:%d", id, j), time.Minute)
                } else {
                    c.Get(ctx, key)
                }
            }
        }(i)
    }
    
    wg.Wait()
    stats, _ := c.Stats(ctx)
    fmt.Printf("Completed 100,000 concurrent operations\n")
    fmt.Printf("Hit ratio: %.2f%%\n", float64(stats.Hits)*100/float64(stats.Hits+stats.Misses))
}
```

## 📊 Performance Benchmarks

### Test Environment

All benchmarks were conducted with the following setup:

- **CPU**: AMD Ryzen 5 5600G with Radeon Graphics
- **Memory**: 16GB DDR4-3200
- **OS**: Windows 10
- **Go Version**: 1.18+
- **Test Duration**: Each benchmark repeated 3 times, each run for 3 seconds

### Core Operations Performance

| Operation | Cache Size | Performance (ns/op) | Memory (B/op) | Allocations (allocs/op) |
|-----------|------------|---------------------|---------------|-------------------------|
| Get/Hit | 1,000 | 97.47 | 0 | 0 |
| Get/Hit | 10,000 | 97.31 | 0 | 0 |
| Get/Hit | 100,000 | 98.98 | 0 | 0 |
| Get/Miss | 1,000 | 128.33 | 24 | 2 |
| Get/Miss | 10,000 | 129.30 | 24 | 1 |
| Get/Miss | 100,000 | 123.87 | 24 | 1 |
| Set/New | 1,000 | 439.13 | 72.7 | 3 |
| Set/New | 10,000 | 442.03 | 72.3 | 3 |
| Set/New | 100,000 | 456.80 | 70.7 | 3 |
| Set/Existing | 1,000 | 179.03 | 24 | 1 |
| Set/Existing | 10,000 | 162.60 | 24 | 1 |
| Set/Existing | 100,000 | 170.93 | 24 | 1 |
| Mixed (80% Read/20% Write) | 1,000 | 249.17 | 4 | 0 |
| Mixed (80% Read/20% Write) | 10,000 | 258.70 | 4 | 0 |
| Mixed (80% Read/20% Write) | 100,000 | 332.00 | 4 | 0 |

**Analysis**:
- **Get/Hit Performance**: Extremely efficient at ~97-99ns with zero memory allocations
- **Get/Miss Performance**: Still very fast at ~123-129ns with minimal allocations
- **Set Operations**: New entries take ~440-457ns, while updating existing entries is much faster at ~163-179ns
- **Cache Size Impact**: Performance remains stable across different cache sizes, with only slight degradation at 100,000 entries
- **Memory Efficiency**: Read operations (Get/Hit) allocate no memory, demonstrating excellent efficiency

### Eviction Policies Comparison

| Eviction Policy | Performance (ns/op) | Memory (B/op) | Allocations (allocs/op) | Ranking |
|-----------------|---------------------|---------------|-------------------------|---------|
| LRU | 123.60 | 12 | 0 | 4 |
| LFU | 119.03 | 12 | 0 | 3 |
| FIFO | 119.00 | 12 | 0 | 1 |
| Random | 118.83 | 12 | 0 | 2 |

**Analysis**:
- All eviction policies perform similarly, with differences under 5%
- Random and FIFO are slightly faster due to simpler decision logic
- LRU has marginally higher overhead due to recency tracking
- All policies show excellent memory efficiency with the same allocation profile
- LFU's memory usage is well-optimized, showing no additional overhead compared to simpler policies

### Concurrency Performance

Stress tests were conducted with different concurrency levels and read/write ratios:

| Scenario | Threads | QPS | Read/Write | Success Rate | Max Latency | Hit Rate |
|----------|---------|-----|------------|--------------|-------------|----------|
| Standard Load | 4 | 1000 | 80%/20% | 100% | 1.00ms | 24.38% |
| High Concurrency | 8 | 2000 | 80%/20% | 100% | 0.48ms | 37.46% |
| Write-Intensive | 8 | 2000 | 20%/80% | 100% | 0.44ms | 77.43% |

**Analysis**:
- HCache maintains 100% success rate even under high load
- Latency remains sub-millisecond in all scenarios
- Higher concurrency actually shows lower max latency, demonstrating effective sharding
- Write-intensive workloads achieve higher hit rates due to faster cache population
- The cache scales linearly with increased QPS and thread count

### Memory Efficiency

Memory usage analysis for different eviction policies with 100,000 entries:

| Eviction Policy | Per-Entry Metadata (bytes) | Total Metadata for 100K entries | Relative Overhead |
|-----------------|----------------------------|----------------------------------|------------------|
| LRU | ~24 | ~2.3 MB | 100% |
| LFU | ~16 | ~1.5 MB | 65% |
| FIFO | ~16 | ~1.5 MB | 65% |
| Random | ~8 | ~0.8 MB | 35% |

**Analysis**:
- Random policy has the lowest memory overhead
- LRU requires more metadata to track access recency
- LFU is efficiently implemented with overhead comparable to FIFO
- For small caches, these differences are negligible
- For very large caches (millions of entries), policy selection can significantly impact memory usage

## 👥 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please make sure to update tests as appropriate.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 