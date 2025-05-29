# HCache - High-Performance Go Cache Library

HCache is a high-performance, feature-rich in-memory cache library for Go applications, designed with a focus on performance, concurrency, and flexibility.

## Features

- **High Performance**: Optimized for multi-core systems with sharded design
- **Flexible Eviction Policies**: LRU, LFU, FIFO, and Random eviction strategies
- **TTL Support**: Automatic expiration of cache entries
- **Metrics Collection**: Detailed performance metrics for monitoring
- **Admission Control**: Prevent cache thrashing with intelligent admission policies
- **Concurrency Safe**: Thread-safe operations with minimal lock contention
- **Memory Bounded**: Configurable memory limits to prevent OOM situations
- **Serialization Support**: Pluggable codecs for value serialization
- **Data Loading**: Built-in support for data loaders and cache-aside pattern

## Installation

```bash
go get github.com/yourusername/hcache
```

## Quick Start

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

	// Delete a value
	deleted, err := c.Delete(ctx, "key1")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Deleted: %v\n", deleted)
}
```

## Configuration

HCache can be configured through code or configuration files:

```go
// Configure through code
config := cache.NewDefaultConfig().
	WithName("myCache").
	WithMaxEntries(10000).
	WithMaxMemory(100 * 1024 * 1024). // 100 MB
	WithDefaultTTL(time.Hour).
	WithShardCount(16).
	WithEvictionPolicy("lru")

cache, err := cache.New(config)
```

```yaml
# config.yaml
name: myCache
max_entries: 10000
max_memory_bytes: 104857600  # 100 MB
default_ttl: 3600s
shard_count: 16
eviction_policy: lru
enable_metrics: true
metrics_level: basic
enable_admission_policy: false
cleanup_interval: 60s
enable_compression: false
compression_threshold: 4096
enable_sharded_lock: true
```

```go
// Load from YAML file
cache, err := cache.NewFromFile("config.yaml")
```

## Advanced Usage

### Using Data Loaders

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/hcache/pkg/cache"
)

func main() {
	c, _ := cache.NewWithOptions("myCache")
	defer c.Close()

	ctx := context.Background()

	// Get with loader (will populate cache if key doesn't exist)
	value, err := c.GetWithLoader(ctx, "user:123", func(ctx context.Context) (interface{}, time.Duration, error) {
		// Simulate database lookup
		time.Sleep(100 * time.Millisecond)
		return map[string]string{"name": "John", "email": "john@example.com"}, time.Minute * 5, nil
	})

	if err != nil {
		panic(err)
	}

	fmt.Printf("User: %v\n", value)
}
```

### Using Codecs

```go
package main

import (
	"context"
	"fmt"

	"github.com/yourusername/hcache/pkg/cache"
	"github.com/yourusername/hcache/pkg/codec"
)

type User struct {
	ID    int
	Name  string
	Email string
}

func main() {
	// Create cache with custom codec
	config := cache.NewDefaultConfig()
	config.Codec = codec.NewGobCodec() // Use Gob serialization

	c, _ := cache.New(config)
	defer c.Close()

	ctx := context.Background()

	// Store a struct
	user := User{ID: 1, Name: "John", Email: "john@example.com"}
	c.Set(ctx, "user:1", user, 0)

	// Retrieve the struct
	var retrievedUser User
	value, exists, _ := c.Get(ctx, "user:1")
	if exists {
		retrievedUser = value.(User)
		fmt.Printf("User: %+v\n", retrievedUser)
	}
}
```

## Performance Tips

1. **Choose the right shard count**: Set it to at least the number of CPU cores
2. **Use appropriate TTLs**: Avoid indefinite caching when data can become stale
3. **Monitor metrics**: Watch hit ratios and adjust cache size accordingly
4. **Consider admission policy**: Enable for high-traffic caches with skewed access patterns
5. **Benchmark your workload**: Test different configurations for your specific use case

## License

MIT License - see LICENSE file for details. 