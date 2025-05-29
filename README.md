# HCache

<div align="center">

[![Go Reference](https://pkg.go.dev/badge/github.com/yourusername/hcache.svg)](https://pkg.go.dev/github.com/yourusername/hcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/hcache)](https://goreportcard.com/report/github.com/yourusername/hcache)
[![License](https://img.shields.io/github/license/yourusername/hcache)](LICENSE)
[![Build Status](https://github.com/yourusername/hcache/workflows/build/badge.svg)](https://github.com/yourusername/hcache/actions)
[![Coverage](https://codecov.io/gh/yourusername/hcache/branch/main/graph/badge.svg)](https://codecov.io/gh/yourusername/hcache)

<p>A high-performance, feature-rich in-memory cache library for Go applications</p>
</div>

## 📋 Table of Contents

- [Features](#-features)
- [Installation](#-installation)
- [Quick Start](#-quick-start)
- [Usage Examples](#-usage-examples)
  - [Basic Operations](#basic-operations)
  - [Cache-Aside Pattern](#cache-aside-pattern)
  - [HTTP Server Integration](#http-server-integration)
- [Architecture](#-architecture)
- [Performance](#-performance)
- [Configuration](#-configuration)
- [Advanced Features](#-advanced-features)
- [Contributing](#-contributing)
- [License](#-license)

## ✨ Features

- **High Performance**: Optimized for multi-core systems with sharded design for minimal lock contention
- **Flexible Eviction Policies**: Supports LRU, LFU, FIFO, and Random eviction strategies
- **TTL Support**: Automatic expiration of cache entries with customizable time-to-live
- **Metrics Collection**: Detailed performance metrics for monitoring cache efficiency
- **Admission Control**: Prevents cache thrashing with intelligent admission policies
- **Concurrency Safe**: Thread-safe operations for reliable use in concurrent environments
- **Memory Bounded**: Configurable memory limits to prevent out-of-memory situations
- **Serialization Support**: Pluggable codecs for value serialization and compression
- **Data Loading**: Built-in support for data loaders and cache-aside pattern
- **Extensible**: Modular design allows for custom implementations of core components

## 📦 Installation

```bash
go get github.com/yourusername/hcache
```

## 🚀 Quick Start

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/hcache/pkg/cache"
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

## 📝 Usage Examples

### Basic Operations

```go
// Set with TTL
cache.Set(ctx, "user:1001", userData, time.Hour)

// Get
value, exists, err := cache.Get(ctx, "user:1001")

// Delete
removed, err := cache.Delete(ctx, "user:1001")

// Clear all entries
cache.Clear(ctx)

// Get stats
stats, err := cache.Stats(ctx)
fmt.Printf("Hits: %d, Misses: %d, Ratio: %.2f%%\n", 
           stats.Hits, stats.Misses, stats.HitRatio*100)
```

### Cache-Aside Pattern

```go
import "github.com/yourusername/hcache/pkg/loader"

// Create a data loader
userLoader := loader.NewFunctionLoader(func(ctx context.Context, key string) (interface{}, error) {
    // Fetch data from database when not in cache
    return fetchUserFromDatabase(key)
})

// Create cache with loader
c, err := cache.NewWithOptions("userCache",
    cache.WithMaxEntryCount(10000),
    cache.WithLoader(userLoader),
    cache.WithTTL(time.Hour),
)

// Get or load data
userData, err := c.GetOrLoad(ctx, "user:1001")
```

### HTTP Server Integration

See the [examples/http_server](examples/http_server) directory for a complete example of integrating HCache with an HTTP server.

## 🏗️ Architecture

HCache is built with a layered architecture designed for performance, flexibility, and extensibility:

- **pkg/**: Public API and interfaces for external use
  - **cache/**: Main cache interface and implementation
  - **loader/**: Data loading interfaces for cache misses
  - **codec/**: Serialization interfaces and implementations
  - **errors/**: Standardized error types
  
- **internal/**: Implementation details (not for external use)
  - **metrics/**: Performance metrics collection
  - **storage/**: Internal data storage mechanisms
  - **eviction/**: Eviction policy implementations
  - **ttl/**: Time-to-live management
  - **admission/**: Admission policy implementations
  - **utils/**: Utility functions and data structures

## 📊 Performance

HCache is designed for high performance in multi-core environments. The benchmark suite covers various scenarios including different:

- Cache sizes
- Concurrency levels
- Access patterns (including Zipfian distribution)
- Value sizes
- Read/write ratios
- Eviction policies

Run benchmarks with:

```bash
# Linux/macOS
./test/run_benchmarks.sh

# Windows
./test/run_benchmarks.ps1
```

Sample benchmark results (Intel Core i7, 16GB RAM):

| Benchmark | Operations | ns/op | B/op | allocs/op |
|-----------|------------|-------|------|-----------|
| Get/Hit | 20000000 | 63.1 | 8 | 1 |
| Get/Miss | 10000000 | 115.0 | 24 | 2 |
| Set/New | 5000000 | 235.0 | 40 | 3 |
| Set/Existing | 5000000 | 210.0 | 32 | 2 |
| Mixed/Read80Write20 | 5000000 | 180.0 | 32 | 2 |

## ⚙️ Configuration

HCache provides a flexible configuration system using functional options:

```go
cache, err := cache.NewWithOptions("myCache",
    // Basic settings
    cache.WithMaxEntryCount(100000),          // Maximum number of entries
    cache.WithMaxMemoryBytes(500*1024*1024),  // Memory limit (500MB)
    cache.WithShards(256),                    // Number of shards for concurrency
    
    // Eviction settings
    cache.WithEviction("lru"),                // LRU, LFU, FIFO, or Random
    cache.WithTTL(time.Hour),                 // Default TTL
    
    // Advanced settings
    cache.WithMetricsEnabled(true),           // Enable metrics collection
    cache.WithAdmissionPolicy(myPolicy),      // Custom admission policy
    cache.WithCodec(myCodec),                 // Custom serialization
    cache.WithLoader(myLoader),               // Data loader for cache misses
)
```

## 🔧 Advanced Features

### Custom Serialization

```go
import "github.com/yourusername/hcache/pkg/codec"

// Create a custom codec
myCodec := codec.NewJSONCodec()

// Use codec with cache
c, err := cache.NewWithOptions("myCache",
    cache.WithCodec(myCodec),
)
```

### Custom Admission Policy

```go
import "github.com/yourusername/hcache/pkg/admission"

// Create a custom admission policy
myPolicy := admission.NewTinyLFU(10000)

// Use admission policy with cache
c, err := cache.NewWithOptions("myCache",
    cache.WithAdmissionPolicy(myPolicy),
)
```

### Metrics Collection

```go
// Enable metrics
c, err := cache.NewWithOptions("myCache",
    cache.WithMetricsEnabled(true),
)

// Get metrics
stats, err := c.Stats(ctx)
fmt.Printf("Hit ratio: %.2f%%\n", stats.HitRatio*100)
fmt.Printf("Evictions: %d\n", stats.Evictions)
fmt.Printf("Average lookup time: %v\n", stats.AverageLookupTime)
```

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