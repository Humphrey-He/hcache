
# hcache/internal Module Technical Manual

![hcache Architecture](https://example.com/hcache-internal-arch.png)
*Figure: hcache Internal Module Architecture*

## Table of Contents
- [Core Positioning](#core-positioning)
- [Module Architecture](#module-architecture)
- [Quick Start](#quick-start)
- [In-depth Guide](#in-depth-guide)
  - [ebpf Real-time Tracking](#ebpf-real-time-tracking)
  - [procfs High-performance Parsing](#procfs-high-performance-parsing)
  - [cache_policy Strategy Simulation](#cache_policy-strategy-simulation)
- [Performance Benchmarks](#performance-benchmarks)
- [Evolution Roadmap](#evolution-roadmap)
- [Contribution Guidelines](#contribution-guidelines)

---

## Core Positioning
`internal` serves as the advanced functional core layer for system-level cache analysis with:
- 🚀 **Kernel-level Observation**: Nanosecond cache event capture via eBPF
- ⚡ **Zero-overhead Collection**: mmap-accelerated /proc file parsing engine
- 🧠 **Intelligent Prediction**: LRU/ARC algorithm simulator

> 💡 Design Philosophy: _"Observation accuracy should not become a performance bottleneck"_

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

## Module Architecture
```
bash
internal/
├── ebpf/               # Kernel-level event tracing
│   ├── tracer.c        # eBPF bytecode
│   └── event_parser.go # User-space decoder
├── procfs/             # /proc parsing engine
│   ├── mmap_reader.go  # Zero-copy reader
│   └── consistency.go  # Optimistic locking mechanism
└── cache_policy/       # Strategy simulation
    ├── arc/            # Adaptive Replacement Cache
    └── simulator.go    # Workload replay tool
```

---

## Quick Start
### Prerequisites
- Linux kernel ≥ 5.8 (eBPF support)
- Go 1.18+

### Basic Usage
```go
import "github.com/Humphrey-He/hcache/internal/procfs"

// Example: Fast process cache scan
scanner := procfs.NewScanner(procfs.WithMMAP(true))
results, _ := scanner.ScanPID(1234) 
fmt.Printf("Cached Pages: %d\n", results[0].Cached)
```

### Enable eBPF Monitoring
```bash
make build-ebpf  # Compile kernel module
hcache --internal=ebpf --pid $(pgrep nginx)
```

## In-depth Guide
### ebpf Real-time Tracking
#### Technical Implementation
``c
// Kernel module example (tracer.c)
SEC("kprobe/__page_cache_alloc")
int trace_cache_alloc(struct pt_regs *ctx) {
    u32 pid = bpf_get_current_pid_tgid();
    bpf_perf_event_output(ctx, &events, BPF_F_CURRENT_CPU, &pid, sizeof(pid));
    return 0;
}
```
#### Typical Output
```
TIME               PID    FILE                  HIT/MISS
14:00:01.1234      4567   /var/log/nginx.log    MISS
14:00:01.1235      4567   /lib/libc.so.6       HIT
```

### procfs High-performance Parsing
#### Performance Comparison
| Method          | Time for 100k files | CPU Usage |
|-----------------|---------------------|-----------|
| Traditional line-by-line reading | 4.2s         | 12%       |
| mmap+unsafe     | 0.5s                | 3%        |

### cache_policy Strategy Simulation
```go
// ARC algorithm core logic
func (a *ARC) Access(key string) {
    if a.t1.Contains(key) {
        a.t2.PushFront(a.t1.Remove(key))
    }
    // ... Adaptive adjustment logic
}
```


## Performance Benchmarks
### Test Environment
- AWS c5.4xlarge (16 vCPU)
- Linux 5.15.0

### Throughput Testing
```bash
$ make bench
BenchmarkProcfsScan-16   	  150000	      7852 ns/op  # 127k QPS
BenchmarkEBPFEvent-16    	 2000000	       901 ns/op  # 1.1M events/sec
```

---

## Evolution Roadmap
| Version | Milestone                          | ETA     |
|--------|------------------------------------|---------|
| v1.0   | Basic eBPF tracing framework       | 2023Q4  |
| v2.0   | Multi-dimensional cache heatmap    | 2024Q1  |
| v3.0   | LSTM-based prediction model        | 2024Q3  |

---

## Contribution Guidelines
We welcome your contributions! Please follow:
1. Run `make verify` to pass all checks before submission
2. New eBPF code needs kernel version compatibility testing
3. Performance optimizations need benchmark comparison

```bash
# Development environment setup
git clone https://github.com/Humphrey-He/hcache
cd hcache/internal
make dev-env 
```

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
