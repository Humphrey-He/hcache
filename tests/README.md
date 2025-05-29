# HCache 测试框架

本目录包含 HCache 的三大核心测试能力：基准测试、并发压测和命中率测试。这些测试旨在全面评估 HCache 的性能、稳定性和实际应用场景下的效果。

## 测试类型

### 1. 基准性能测试 (Benchmark)

位于 `benchmark/` 目录，使用 Go 标准的基准测试框架，针对 HCache 的核心函数进行性能测试。

**主要指标**：
- **ns/op**: 每次操作的平均耗时（纳秒）
- **B/op**: 每次操作的内存分配量（字节）
- **allocs/op**: 每次操作的内存分配次数

**运行方式**：
```bash
cd tests/benchmark
go test -bench=. -benchmem
```

**高级用法**：
```bash
# 运行特定测试，并保存结果
go test -bench=BenchmarkGet -benchmem > result/get_$(date +%Y%m%d).txt

# 比较两次测试结果
benchstat result/get_20220101.txt result/get_20220201.txt
```

### 2. 高并发压测 (Concurrency)

位于 `concurrency/` 目录，使用 vegeta 工具模拟高并发场景下的 HCache 性能。

**主要指标**：
- **P50/P95/P99 延迟**: 不同百分位的请求延迟
- **吞吐量 (RPS)**: 每秒处理的请求数
- **错误率**: 请求失败的比例
- **GC 情况**: 垃圾回收次数和内存增长曲线

**运行方式**：
```bash
# 启动测试服务器
cd concurrency/mock_server
go run main.go

# 在另一个终端运行压测
cd concurrency/vegeta
./attack.sh
```

### 3. 命中率测试 (Hit Ratio Tests)

命中率测试位于 `tests/hitratio` 目录中，用于评估不同访问模式下缓存的命中率性能。

#### 测试类型

##### 1. 基本分布测试 (Basic Distribution Tests)

- **均匀分布 (Uniform Distribution)**: 键以均匀概率被访问
- **Zipf 分布 (Zipf Distribution)**: 键按照 Zipf 定律（幂律分布）被访问，模拟真实世界的缓存访问模式
  - 低偏斜 (Low Skew): s=1.07
  - 高偏斜 (High Skew): s=1.2

##### 2. 专业访问模式测试 (Specialized Access Pattern Tests)

- **竞争抵抗测试 (Contention Resistance Test)**: 测试缓存在高竞争条件下的性能，即多种访问模式竞争相同缓存空间的情况
- **搜索模式测试 (Search Pattern Test)**: 模拟搜索引擎查询模式，少量热门词汇和大量罕见词汇
- **数据库模式测试 (Database Pattern Test)**: 模拟数据库访问模式，包括记录访问和索引查找
- **循环模式测试 (Looping Pattern Test)**: 模拟循环访问模式，相同的数据集被重复访问
- **CODASYL模式测试 (CODASYL Pattern Test)**: 模拟网络数据库模式，数据在图结构中被访问

#### 运行测试

```bash
# 运行所有命中率测试
go test -v ./tests/hitratio/...

# 运行特定测试
go test -v ./tests/hitratio -run "TestContentionResistance"
go test -v ./tests/hitratio -run "TestSearchPattern"
go test -v ./tests/hitratio -run "TestDatabasePattern"
go test -v ./tests/hitratio -run "TestLoopingPattern"
go test -v ./tests/hitratio -run "TestCODASYLPattern"
```

#### 自动化测试和可视化

我们提供了自动化脚本来运行测试并生成可视化结果:

```bash
# Windows PowerShell
./tests/run_hitratio_tests.ps1

# 可视化结果 (需要 Python 环境)
cd tests/analysis
pip install -r requirements.txt
python hitratio_visualizer.py
```

可视化结果将保存在 `results/hitratio/visualizations` 目录中，包括:
- 条形图: 每个测试模式的命中率对比
- 策略比较图: 不同策略在各测试模式下的表现
- 热图: 所有测试模式和策略的命中率矩阵
- 雷达图: 策略在各测试模式下的性能比较

## 性能分析工具包 (Analysis Toolkit)

位于 `analysis/` 目录，提供了一套完整的性能分析工具，用于处理和可视化测试结果。

**主要功能**：
- **基准测试分析**: 处理 Go 基准测试结果，分析延迟、内存分配和吞吐量
- **命中率分析**: 分析不同淘汰策略和访问模式下的缓存命中率
- **并发性能分析**: 评估不同并发级别下的性能
- **pprof 分析**: 生成火焰图并从 Go pprof 配置文件中识别热点
- **综合分析报告**: 将所有分析结合到一个全面的报告中

**运行方式**：
```bash
# 在 Linux/Mac 上运行
cd tests
./run_and_analyze.sh

# 在 Windows 上运行
cd tests
.\run_and_analyze.ps1
```

**分析工具的输出**：
- **CSV 文件**: 原始数据和统计信息
- **图像**: 静态图表和图形
- **HTML 文件**: 使用 Plotly 的交互式可视化
- **Markdown 报告**: 详细的分析报告
- **HTML 报告**: 报告的交互式 HTML 版本
- **Excel 摘要**: 关键指标汇总
- **火焰图**: CPU 和内存配置文件的 SVG 火焰图

**要求**：
- Python 3.8+
- 必要的 Python 包（通过 `pip install -r analysis/requirements.txt` 安装）
- Go 工具链（用于 pprof 分析）

## 测试结果汇总

| 测试类型 | 指标 | 描述说明 |
|---------|------|---------|
| Benchmark | ns/op | 平均操作耗时 |
|           | allocs/op | 每操作分配次数 |
|           | B/op | 平均内存分配字节数 |
| Concurrency | P95 Latency | 压测延迟关键指标 |
|             | RPS | 请求吞吐量 |
|             | Success Rate | 请求成功比率 |
| HitRatio | Hit Rate (%) | 缓存命中率 |
|          | Eviction Rate | 淘汰比率 |
|          | Access Skew Impact | 高热点分布场景对命中率影响（Zipf 模拟）| 