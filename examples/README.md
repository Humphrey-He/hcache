# HCache Examples

This directory contains examples and benchmarks that demonstrate how to use the HCache library in various scenarios.

## HTTP Server Example

The `http_server` directory contains a complete example of integrating HCache with a Gin-based web server for an e-commerce product API.

### Features

- Layered architecture (handler, service, storage)
- Cache integration with proper error handling
- Cache preloading for popular products
- Cache invalidation on updates
- Metrics collection and reporting
- Middleware for request logging and cache stats

### Running the example

```bash
cd examples/http_server
go run main.go
```

Options:
```
  -cache-size int
        Maximum number of cache entries (default 1000)
  -port string
        HTTP server port (default "8080")
  -shards int
        Number of cache shards (default 16)
  -ttl duration
        Default TTL for cache entries (default 1m0s)
```

### API Endpoints

- `GET /products/:id` - Get a single product by ID
- `GET /products` - List products with optional filtering
- `POST /products` - Create a new product
- `PUT /products/:id` - Update an existing product
- `DELETE /products/:id` - Delete a product
- `GET /cache/stats` - Get cache statistics

## Stress Test

The `stress_test` directory contains a tool for stress testing the cache under various workloads.

### Features

- Configurable QPS, duration, and concurrency
- Adjustable read/write ratio
- Real-time metrics reporting
- CSV and Markdown output formats
- Support for different key distributions

### Running the stress test

```bash
cd examples/stress_test
go run main.go
```

Options:
```
  -cache-size int
        Maximum number of cache entries (default 100000)
  -duration duration
        Test duration (default 30s)
  -keys int
        Number of unique keys to use (default 10000)
  -output string
        Output format (text, csv, markdown) (default "text")
  -qps int
        Target QPS (default 1000)
  -read-pct int
        Percentage of read operations (vs writes) (default 80)
  -report-interval duration
        Interval for reporting stats (default 1s)
  -shards int
        Number of cache shards (default 16)
  -ttl duration
        Default TTL for cache entries (default 5m0s)
  -value-size int
        Size of values in bytes (default 1024)
  -workers int
        Number of concurrent workers (default 10)
```

## Benchmarks

The `benchmark` directory contains Go benchmarks for measuring the performance of different cache operations.

### Features

- Benchmarks for different cache configurations (size, shards)
- Tests for different access patterns (read-heavy, write-heavy, mixed)
- Realistic workload simulations with zipfian distributions
- Eviction policy comparisons
- TTL expiration benchmarks

### Running the benchmarks

```bash
cd examples/benchmark
go test -bench=. -benchmem
```

For more detailed output:

```bash
go test -bench=. -benchmem -benchtime=5s -count=3
```

To run a specific benchmark:

```bash
go test -bench=BenchmarkCache/Size=10000/Shards=16 -benchmem
``` 