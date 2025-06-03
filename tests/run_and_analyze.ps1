# PowerShell script to run all HCache tests and analyze the results
# PowerShell 脚本，用于运行所有 HCache 测试并分析结果

<#
.DESCRIPTION
    This script automates the execution of performance tests for HCache and analyzes the results.
    It includes benchmark tests, hit ratio tests, concurrency tests, and profile analysis.
    
    此脚本自动执行 HCache 的性能测试并分析结果。
    包括基准测试、命中率测试、并发测试和性能分析。

.PARAMETERS
    No direct parameters, but the script offers interactive choices during execution.
    
    没有直接参数，但脚本在执行过程中提供交互式选择。

.TEST TARGETS
    - Benchmark: Measures operation latency, memory allocation, and allocations per operation
      基准测试：测量操作延迟、内存分配和每操作分配次数
    
    - Hit Ratio: Evaluates cache efficiency with different eviction policies and access patterns
      命中率：评估不同淘汰策略和访问模式下的缓存效率
    
    - Concurrency: Tests performance under various concurrent loads (1-200 concurrent users)
      并发：测试在不同并发负载下的性能（1-200个并发用户）
    
    - pprof: CPU and memory profiling to identify bottlenecks
      性能分析：CPU和内存分析，用于识别瓶颈
#>

# Set the base directory to the script's location
# 将基础目录设置为脚本所在位置
$BASE_DIR = $PSScriptRoot
$RESULTS_DIR = Join-Path $BASE_DIR "results"
$ANALYSIS_DIR = Join-Path $RESULTS_DIR "analysis"

# Create results directories
# 创建结果目录
New-Item -Path (Join-Path $RESULTS_DIR "benchmark") -ItemType Directory -Force | Out-Null
New-Item -Path (Join-Path $RESULTS_DIR "hitratio") -ItemType Directory -Force | Out-Null
New-Item -Path (Join-Path $RESULTS_DIR "concurrency") -ItemType Directory -Force | Out-Null
New-Item -Path (Join-Path $RESULTS_DIR "pprof") -ItemType Directory -Force | Out-Null

# Function to display section headers
# 显示章节标题的函数
function Show-Section {
    param (
        [string]$Title
    )
    
    Write-Host ""
    Write-Host "=====================================================" -ForegroundColor Cyan
    Write-Host "  $Title" -ForegroundColor Cyan
    Write-Host "=====================================================" -ForegroundColor Cyan
    Write-Host ""
}

# Check if Python and required packages are installed
# 检查是否安装了Python和所需的包
function Test-PythonDeps {
    Show-Section "Checking Python dependencies"
    
    try {
        $pythonVersion = python --version 2>&1
        Write-Host "Python installed: $pythonVersion"
    }
    catch {
        Write-Host "Python is not installed. Please install Python 3.8 or higher." -ForegroundColor Red
        exit 1
    }
    
    try {
        python -c "import pandas, numpy, matplotlib, seaborn, plotly" 2>&1 | Out-Null
        Write-Host "Python dependencies are already installed."
    }
    catch {
        Write-Host "Installing required Python packages..."
        pip install -r (Join-Path $BASE_DIR "analysis\requirements.txt")
    }
}

# Run benchmark tests
# 运行基准测试
function Start-BenchmarkTests {
    <#
    .DESCRIPTION
        Executes Go benchmark tests to measure core operations performance.
        执行Go基准测试，测量核心操作性能。
        
    .TEST PARAMETERS
        - bench: Runs all benchmarks (.)
          bench参数：运行所有基准测试 (.)
        
        - benchmem: Includes memory allocation statistics
          benchmem参数：包含内存分配统计
        
        - count=5: Runs each benchmark 5 times for statistical stability
          count=5参数：每个基准测试运行5次以保证统计稳定性
        
    .TEST METRICS
        - ns/op: Nanoseconds per operation (lower is better)
          ns/op：每操作纳秒数（越低越好）
        
        - B/op: Bytes allocated per operation (lower is better)
          B/op：每操作分配字节数（越低越好）
        
        - allocs/op: Number of heap allocations per operation (lower is better)
          allocs/op：每操作堆分配次数（越低越好）
    #>
    Show-Section "Running benchmark tests"
    
    Push-Location (Join-Path $BASE_DIR "benchmark")
    
    # Enable CPU profiling
    # 启用CPU分析
    $env:CPUPROFILE = Join-Path $RESULTS_DIR "pprof\benchmark_cpu_$(Get-Date -Format 'yyyyMMdd').pprof"
    $env:MEMPROFILE = Join-Path $RESULTS_DIR "pprof\benchmark_mem_$(Get-Date -Format 'yyyyMMdd').pprof"
    
    Write-Host "Running benchmark tests..."
    $benchmarkOutput = Join-Path $RESULTS_DIR "benchmark\benchmark_$(Get-Date -Format 'yyyyMMdd').txt"
    go test -bench=. -benchmem -count=5 | Tee-Object -FilePath $benchmarkOutput
    
    Write-Host "Benchmark tests completed."
    
    Pop-Location
}

# Run hit ratio tests
# 运行命中率测试
function Start-HitRatioTests {
    <#
    .DESCRIPTION
        Executes cache hit ratio tests with different eviction policies and access patterns.
        使用不同的淘汰策略和访问模式执行缓存命中率测试。
        
    .TEST PARAMETERS
        - v: Verbose output showing test details
          v参数：详细输出显示测试细节
        
    .TEST SCENARIOS
        - Access patterns: Uniform, Zipfian, Looping, Database-like, Search patterns
          访问模式：均匀分布、Zipf分布、循环访问、数据库类型、搜索模式
        
        - Eviction policies: LRU, LFU, FIFO, Random
          淘汰策略：LRU（最近最少使用）、LFU（最不经常使用）、FIFO（先进先出）、Random（随机）
        
        - Cache sizes: Various sizes to test scaling behavior
          缓存大小：测试不同大小下的扩展行为
    #>
    Show-Section "Running hit ratio tests"
    
    Push-Location (Join-Path $BASE_DIR "hitratio")
    
    Write-Host "Running hit ratio tests..."
    $hitratioOutput = Join-Path $RESULTS_DIR "hitratio\hitratio_$(Get-Date -Format 'yyyyMMdd').txt"
    go test -v | Tee-Object -FilePath $hitratioOutput
    
    Write-Host "Hit ratio tests completed."
    
    Pop-Location
}

# Run concurrency tests
# 运行并发测试
function Start-ConcurrencyTests {
    <#
    .DESCRIPTION
        Tests cache performance under concurrent load using vegeta load testing tool.
        使用vegeta负载测试工具测试并发负载下的缓存性能。
        
    .TEST PARAMETERS
        - rate: Requests per second (varies with concurrency level)
          rate参数：每秒请求数（随并发级别变化）
        
        - duration: Test duration in seconds (30s)
          duration参数：测试持续时间（30秒）
        
        - targets: File containing endpoint URLs to test
          targets参数：包含要测试的端点URL的文件
        
    .CONCURRENCY LEVELS
        Tests with 1, 10, 50, 100, and 200 concurrent users.
        Request rate scales with concurrency (10x multiplier).
        
        测试1、10、50、100和200个并发用户。
        请求速率随并发性扩展（10倍乘数）。
        
    .TEST METRICS
        - Latency: P50, P95, P99 response times
          延迟：P50、P95、P99响应时间
        
        - Throughput: Requests per second achieved
          吞吐量：实现的每秒请求数
        
        - Success rate: Percentage of successful requests
          成功率：成功请求的百分比
    #>
    Show-Section "Running concurrency tests"
    
    Push-Location (Join-Path $BASE_DIR "concurrency")
    
    Write-Host "Starting mock server..."
    Push-Location "mock_server"
    $serverJob = Start-Job -ScriptBlock { 
        Set-Location $using:PWD
        go run main.go 
    }
    
    # Wait for server to start
    # 等待服务器启动
    Start-Sleep -Seconds 2
    
    Write-Host "Running vegeta load tests..."
    Push-Location "..\vegeta"
    
    # Run with different concurrency levels
    # 以不同的并发级别运行
    foreach ($concurrency in @(1, 10, 50, 100, 200)) {
        Write-Host "Running with concurrency $concurrency..."
        
        # Adjust rate based on concurrency
        # 根据并发性调整速率
        $rate = $concurrency * 10
        
        # Run for 30 seconds
        # 运行30秒
        $outputFile = Join-Path $RESULTS_DIR "concurrency\vegeta_c${concurrency}_r${rate}_$(Get-Date -Format 'yyyyMMdd').json"
        Write-Host "vegeta attack -rate=$rate -duration=30s -targets=targets.txt | vegeta report -type=json > $outputFile"
        
        # Check if vegeta is installed
        # 检查是否安装了vegeta
        try {
            vegeta attack -rate=$rate -duration=30s -targets=targets.txt | vegeta report -type=json | Out-File -FilePath $outputFile
        }
        catch {
            Write-Host "Vegeta is not installed. Please install vegeta to run concurrency tests." -ForegroundColor Yellow
            break
        }
    }
    
    # Stop the mock server
    # 停止模拟服务器
    Stop-Job -Job $serverJob
    Remove-Job -Job $serverJob -Force
    
    Write-Host "Concurrency tests completed."
    
    Pop-Location
    Pop-Location
}

# Run pprof analysis
# 运行pprof分析
function Start-PprofAnalysis {
    <#
    .DESCRIPTION
        Analyzes CPU and memory profiles to identify performance bottlenecks.
        分析CPU和内存配置文件，识别性能瓶颈。
        
    .PROFILING TARGETS
        - CPU Profile: Identifies functions consuming the most CPU time
          CPU配置文件：识别消耗最多CPU时间的函数
        
        - Memory Profile: Identifies functions allocating the most memory
          内存配置文件：识别分配最多内存的函数
        
    .ANALYSIS OUTPUT
        - Text reports showing top consumers of CPU and memory
          显示CPU和内存主要消耗者的文本报告
    #>
    Show-Section "Running pprof analysis"
    
    # Check if pprof profiles exist
    # 检查pprof配置文件是否存在
    $cpuProfile = Join-Path $RESULTS_DIR "pprof\benchmark_cpu_$(Get-Date -Format 'yyyyMMdd').pprof"
    if (-not (Test-Path $cpuProfile)) {
        Write-Host "No pprof profiles found. Skipping pprof analysis." -ForegroundColor Yellow
        return
    }
    
    Write-Host "Generating CPU profile summary..."
    $cpuSummary = Join-Path $RESULTS_DIR "pprof\cpu_summary_$(Get-Date -Format 'yyyyMMdd').txt"
    go tool pprof -text $cpuProfile | Out-File -FilePath $cpuSummary
    
    Write-Host "Generating memory profile summary..."
    $memProfile = Join-Path $RESULTS_DIR "pprof\benchmark_mem_$(Get-Date -Format 'yyyyMMdd').pprof"
    $memSummary = Join-Path $RESULTS_DIR "pprof\mem_summary_$(Get-Date -Format 'yyyyMMdd').txt"
    go tool pprof -text $memProfile | Out-File -FilePath $memSummary
    
    Write-Host "pprof analysis completed."
}

# Run analysis tools
# 运行分析工具
function Start-Analysis {
    <#
    .DESCRIPTION
        Runs Python-based analysis tools to process test results and generate visualizations.
        运行基于Python的分析工具，处理测试结果并生成可视化。
        
    .ANALYSIS TARGETS
        - Benchmark results: Performance trends and comparisons
          基准测试结果：性能趋势和比较
        
        - Hit ratio data: Comparison of eviction policies effectiveness
          命中率数据：淘汰策略有效性比较
        
        - Concurrency results: System behavior under load
          并发结果：负载下的系统行为
        
    .OUTPUT FORMATS
        - CSV data files for raw results
          原始结果的CSV数据文件
        
        - Static images (PNG, SVG)
          静态图像（PNG，SVG）
        
        - Interactive HTML visualizations
          交互式HTML可视化
        
        - Markdown and HTML reports
          Markdown和HTML报告
    #>
    Show-Section "Running analysis tools"
    
    Push-Location (Join-Path $BASE_DIR "analysis")
    
    Write-Host "Running comprehensive analysis..."
    python analyze_all.py --base-dir $RESULTS_DIR --output $ANALYSIS_DIR
    
    Write-Host "Analysis completed. Results are available in $ANALYSIS_DIR"
    
    Pop-Location
}

# Main execution
# 主执行函数
function Start-Main {
    Show-Section "HCache Test and Analysis Suite"
    Write-Host "Starting tests and analysis at $(Get-Date)"
    
    # Check dependencies
    # 检查依赖项
    Test-PythonDeps
    
    # Ask user which tests to run
    # 询问用户要运行哪些测试
    Write-Host "Which tests would you like to run?"
    Write-Host "1. All tests"
    Write-Host "2. Benchmark tests only"
    Write-Host "3. Hit ratio tests only"
    Write-Host "4. Concurrency tests only"
    Write-Host "5. Skip tests, run analysis only"
    $choice = Read-Host "Enter your choice (1-5)"
    
    switch ($choice) {
        "1" {
            Start-BenchmarkTests
            Start-HitRatioTests
            Start-ConcurrencyTests
            Start-PprofAnalysis
        }
        "2" {
            Start-BenchmarkTests
            Start-PprofAnalysis
        }
        "3" {
            Start-HitRatioTests
        }
        "4" {
            Start-ConcurrencyTests
        }
        "5" {
            Write-Host "Skipping tests, running analysis only."
        }
        default {
            Write-Host "Invalid choice. Exiting." -ForegroundColor Red
            exit 1
        }
    }
    
    # Run analysis
    # 运行分析
    Start-Analysis
    
    Show-Section "Test and Analysis Completed"
    Write-Host "All tests and analysis completed at $(Get-Date)"
    Write-Host "Results are available in $RESULTS_DIR"
    Write-Host "Analysis results are available in $ANALYSIS_DIR"
}

# Run the main function
# 运行主函数
Start-Main 