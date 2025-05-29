# HCache - High-Performance Go Cache

HCache is a high-performance, thread-safe in-memory cache for Go applications. It provides a simple API with advanced features like TTL expiration, eviction policies, and metrics collection.

## Features

- Thread-safe concurrent access
- Multiple eviction policies (LRU, LFU, FIFO, Random)
- TTL expiration
- Sharded design for better performance
- Metrics collection
- Configurable via YAML/JSON
- Hot reloading of configuration

## Directory Structure

- `configs/`: Configuration files and utilities
- `examples/`: Example applications using HCache
- `internal/`: Internal packages (metrics, utils)
- `pkg/`: Public API
- `test/`: Benchmarks and regression tests

## Configuration

HCache can be configured using YAML or JSON files. See the [configs/README.md](configs/README.md) for details.

Example:

```yaml
cache:
  enable: true
  max_entries: 500000
  default_ttl: 300s
  cleanup_interval: 30s
storage:
  engine: "in-memory"
  shard_count: 256
eviction:
  policy: "lfu"
  batch_size: 128
```

## Testing

HCache includes comprehensive benchmarks and regression tests. See the [test/README.md](test/README.md) for details.

Example:

```bash
# Run all tests
go test ./...

# Run benchmarks
go test -bench=. -benchmem ./test

# Run benchmarks with the helper script
cd test
./run_benchmarks.ps1 -Benchmark "BenchmarkCacheOperations" -BenchTime "5s"
```

## Examples

HCache includes several examples:

- HTTP Server: A Gin-based web server with cache integration
- Stress Test: A tool for stress testing the cache
- Benchmarks: Go benchmarks for measuring cache performance

See the [examples/README.md](examples/README.md) for details.

## License

MIT 