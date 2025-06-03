#!/bin/bash
# Script to run all HCache tests and analyze the results
# 运行所有 HCache 测试并分析结果的脚本

# Set the base directory to the script's location
# 将基础目录设置为脚本所在位置
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$BASE_DIR/results"
ANALYSIS_DIR="$RESULTS_DIR/analysis"

# Create results directories
# 创建结果目录
mkdir -p "$RESULTS_DIR/benchmark"
mkdir -p "$RESULTS_DIR/hitratio"
mkdir -p "$RESULTS_DIR/concurrency"
mkdir -p "$RESULTS_DIR/pprof"

# Function to display section headers
# 显示章节标题的函数
section() {
    echo ""
    echo "====================================================="
    echo "  $1"
    echo "====================================================="
    echo ""
}

# Check if Python and required packages are installed
# 检查是否安装了Python和所需的包
check_python_deps() {
    section "Checking Python dependencies"
    
    if ! command -v python3 &> /dev/null; then
        echo "Python 3 is not installed. Please install Python 3.8 or higher."
        exit 1
    fi
    
    if ! python3 -c "import pandas, numpy, matplotlib, seaborn, plotly" &> /dev/null; then
        echo "Installing required Python packages..."
        pip install -r "$BASE_DIR/analysis/requirements.txt"
    else
        echo "Python dependencies are already installed."
    fi
}

# Run benchmark tests
# 运行基准测试
run_benchmarks() {
    # 描述：执行Go基准测试，测量核心操作性能
    # Description: Executes Go benchmark tests to measure core operations performance
    #
    # 测试参数 / Test Parameters:
    # - bench=.    : 运行所有基准测试 / Runs all benchmarks
    # - benchmem   : 包含内存分配统计 / Includes memory allocation statistics
    # - count=5    : 每个基准测试运行5次以保证统计稳定性 / Runs each benchmark 5 times for statistical stability
    #
    # 测试指标 / Test Metrics:
    # - ns/op      : 每操作纳秒数（越低越好） / Nanoseconds per operation (lower is better)
    # - B/op       : 每操作分配字节数（越低越好） / Bytes allocated per operation (lower is better)
    # - allocs/op  : 每操作堆分配次数（越低越好） / Number of heap allocations per operation (lower is better)
    
    section "Running benchmark tests"
    
    cd "$BASE_DIR/benchmark"
    
    # Enable CPU profiling
    # 启用CPU和内存分析
    export CPUPROFILE="$RESULTS_DIR/pprof/benchmark_cpu_$(date +%Y%m%d).pprof"
    export MEMPROFILE="$RESULTS_DIR/pprof/benchmark_mem_$(date +%Y%m%d).pprof"
    
    echo "Running benchmark tests..."
    go test -bench=. -benchmem -count=5 | tee "$RESULTS_DIR/benchmark/benchmark_$(date +%Y%m%d).txt"
    
    echo "Benchmark tests completed."
}

# Run hit ratio tests
# 运行命中率测试
run_hitratio_tests() {
    # 描述：使用不同的淘汰策略和访问模式执行缓存命中率测试
    # Description: Executes cache hit ratio tests with different eviction policies and access patterns
    #
    # 测试参数 / Test Parameters:
    # - v          : 详细输出显示测试细节 / Verbose output showing test details
    #
    # 测试场景 / Test Scenarios:
    # - 访问模式 / Access patterns: 
    #   * 均匀分布 / Uniform distribution
    #   * Zipf分布 / Zipf distribution (模拟真实世界访问模式 / simulates real-world access patterns)
    #   * 循环访问 / Looping pattern
    #   * 数据库类型 / Database-like pattern
    #   * 搜索模式 / Search pattern
    #
    # - 淘汰策略 / Eviction policies:
    #   * LRU（最近最少使用） / LRU (Least Recently Used)
    #   * LFU（最不经常使用） / LFU (Least Frequently Used)
    #   * FIFO（先进先出） / FIFO (First In First Out)
    #   * Random（随机） / Random
    
    section "Running hit ratio tests"
    
    cd "$BASE_DIR/hitratio"
    
    echo "Running hit ratio tests..."
    go test -v | tee "$RESULTS_DIR/hitratio/hitratio_$(date +%Y%m%d).txt"
    
    echo "Hit ratio tests completed."
}

# Run concurrency tests
# 运行并发测试
run_concurrency_tests() {
    # 描述：使用vegeta负载测试工具测试并发负载下的缓存性能
    # Description: Tests cache performance under concurrent load using vegeta load testing tool
    #
    # 测试参数 / Test Parameters:
    # - rate       : 每秒请求数（随并发级别变化） / Requests per second (varies with concurrency level)
    # - duration   : 测试持续时间（30秒） / Test duration in seconds (30s)
    # - targets    : 包含要测试的端点URL的文件 / File containing endpoint URLs to test
    #
    # 并发级别 / Concurrency Levels:
    # - 测试1、10、50、100和200个并发用户 / Tests with 1, 10, 50, 100, and 200 concurrent users
    # - 请求速率随并发性扩展（10倍乘数） / Request rate scales with concurrency (10x multiplier)
    #
    # 测试指标 / Test Metrics:
    # - 延迟 / Latency: P50, P95, P99响应时间 / P50, P95, P99 response times
    # - 吞吐量 / Throughput: 实现的每秒请求数 / Requests per second achieved
    # - 成功率 / Success rate: 成功请求的百分比 / Percentage of successful requests
    
    section "Running concurrency tests"
    
    cd "$BASE_DIR/concurrency"
    
    echo "Starting mock server..."
    cd mock_server
    go run main.go &
    SERVER_PID=$!
    
    # Wait for server to start
    # 等待服务器启动
    sleep 2
    
    echo "Running vegeta load tests..."
    cd ../vegeta
    
    # Run with different concurrency levels
    # 以不同的并发级别运行
    for CONCURRENCY in 1 10 50 100 200; do
        echo "Running with concurrency $CONCURRENCY..."
        
        # Adjust rate based on concurrency
        # 根据并发性调整速率
        RATE=$((CONCURRENCY * 10))
        
        # Run for 30 seconds
        # 运行30秒
        echo "vegeta attack -rate=$RATE -duration=30s -targets=targets.txt | vegeta report -type=json > $RESULTS_DIR/concurrency/vegeta_c${CONCURRENCY}_r${RATE}_$(date +%Y%m%d).json"
        vegeta attack -rate=$RATE -duration=30s -targets=targets.txt | vegeta report -type=json > "$RESULTS_DIR/concurrency/vegeta_c${CONCURRENCY}_r${RATE}_$(date +%Y%m%d).json"
    done
    
    # Stop the mock server
    # 停止模拟服务器
    kill $SERVER_PID
    
    echo "Concurrency tests completed."
}

# Run pprof analysis
# 运行pprof分析
run_pprof_analysis() {
    # 描述：分析CPU和内存配置文件，识别性能瓶颈
    # Description: Analyzes CPU and memory profiles to identify performance bottlenecks
    #
    # 分析目标 / Profiling Targets:
    # - CPU配置文件 / CPU Profile: 识别消耗最多CPU时间的函数 / Identifies functions consuming the most CPU time
    # - 内存配置文件 / Memory Profile: 识别分配最多内存的函数 / Identifies functions allocating the most memory
    #
    # 分析输出 / Analysis Output:
    # - 显示CPU和内存主要消耗者的文本报告 / Text reports showing top consumers of CPU and memory
    
    section "Running pprof analysis"
    
    # Check if pprof profiles exist
    # 检查pprof配置文件是否存在
    if [ ! -f "$RESULTS_DIR/pprof/benchmark_cpu_$(date +%Y%m%d).pprof" ]; then
        echo "No pprof profiles found. Skipping pprof analysis."
        return
    fi
    
    echo "Generating CPU profile summary..."
    go tool pprof -text "$RESULTS_DIR/pprof/benchmark_cpu_$(date +%Y%m%d).pprof" > "$RESULTS_DIR/pprof/cpu_summary_$(date +%Y%m%d).txt"
    
    echo "Generating memory profile summary..."
    go tool pprof -text "$RESULTS_DIR/pprof/benchmark_mem_$(date +%Y%m%d).pprof" > "$RESULTS_DIR/pprof/mem_summary_$(date +%Y%m%d).txt"
    
    echo "pprof analysis completed."
}

# Run analysis tools
# 运行分析工具
run_analysis() {
    # 描述：运行基于Python的分析工具，处理测试结果并生成可视化
    # Description: Runs Python-based analysis tools to process test results and generate visualizations
    #
    # 分析目标 / Analysis Targets:
    # - 基准测试结果 / Benchmark results: 性能趋势和比较 / Performance trends and comparisons
    # - 命中率数据 / Hit ratio data: 淘汰策略有效性比较 / Comparison of eviction policies effectiveness
    # - 并发结果 / Concurrency results: 负载下的系统行为 / System behavior under load
    #
    # 输出格式 / Output Formats:
    # - CSV数据文件 / CSV data files: 原始结果 / Raw results
    # - 静态图像 / Static images: PNG, SVG格式 / PNG, SVG formats
    # - 交互式HTML可视化 / Interactive HTML visualizations
    # - Markdown和HTML报告 / Markdown and HTML reports
    
    section "Running analysis tools"
    
    cd "$BASE_DIR/analysis"
    
    echo "Running comprehensive analysis..."
    python3 analyze_all.py --base-dir "$RESULTS_DIR" --output "$ANALYSIS_DIR"
    
    echo "Analysis completed. Results are available in $ANALYSIS_DIR"
}

# Main execution
# 主执行函数
main() {
    section "HCache Test and Analysis Suite"
    echo "Starting tests and analysis at $(date)"
    
    # Check dependencies
    # 检查依赖项
    check_python_deps
    
    # Ask user which tests to run
    # 询问用户要运行哪些测试
    echo "Which tests would you like to run?"
    echo "1. All tests"
    echo "2. Benchmark tests only"
    echo "3. Hit ratio tests only"
    echo "4. Concurrency tests only"
    echo "5. Skip tests, run analysis only"
    read -p "Enter your choice (1-5): " choice
    
    case $choice in
        1)
            run_benchmarks
            run_hitratio_tests
            run_concurrency_tests
            run_pprof_analysis
            ;;
        2)
            run_benchmarks
            run_pprof_analysis
            ;;
        3)
            run_hitratio_tests
            ;;
        4)
            run_concurrency_tests
            ;;
        5)
            echo "Skipping tests, running analysis only."
            ;;
        *)
            echo "Invalid choice. Exiting."
            exit 1
            ;;
    esac
    
    # Run analysis
    # 运行分析
    run_analysis
    
    section "Test and Analysis Completed"
    echo "All tests and analysis completed at $(date)"
    echo "Results are available in $RESULTS_DIR"
    echo "Analysis results are available in $ANALYSIS_DIR"
}

# Run the main function
# 运行主函数
main 