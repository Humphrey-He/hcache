# This script runs benchmarks and generates reports for the HCache library.
# It provides various options to customize benchmark execution and output.
#
# 此脚本运行HCache库的基准测试并生成报告。
# 它提供了各种选项来自定义基准测试执行和输出。

param(
    [string]$Benchmark = ".",        # Pattern to match benchmark names / 匹配基准测试名称的模式
    [string]$BenchTime = "3s",       # Duration to run each benchmark / 运行每个基准测试的持续时间
    [int]$Count = 3,                 # Number of times to run each benchmark / 运行每个基准测试的次数
    [string]$CPU = "1,4,8",          # GOMAXPROCS values to test with / 要测试的GOMAXPROCS值
    [string]$OutputDir = "benchmark_results", # Directory to store results / 存储结果的目录
    [string]$CompareWith = ""        # Previous benchmark results to compare against / 要比较的先前基准测试结果
)

# Create output directory if it doesn't exist
# 如果输出目录不存在，则创建它
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

# Get current date and time for unique filenames
# 获取当前日期和时间以生成唯一的文件名
$Date = Get-Date -Format "yyyyMMdd_HHmmss"
$ResultFile = "$OutputDir\benchmark_$Date.txt"
$JsonFile = "$OutputDir\benchmark_$Date.json"
$MarkdownFile = "$OutputDir\benchmark_$Date.md"
$CpuProfile = "$OutputDir\cpu_$Date.prof"
$MemProfile = "$OutputDir\mem_$Date.prof"

# Print benchmark information before starting
# 在开始前打印基准测试信息
Write-Host "Running benchmarks:"
Write-Host "  Pattern: $Benchmark"
Write-Host "  Duration: $BenchTime"
Write-Host "  Count: $Count"
Write-Host "  CPU: $CPU"
Write-Host "  Output: $ResultFile"
Write-Host ""

# Run benchmarks with the specified parameters
# 使用指定的参数运行基准测试
$benchmarkCommand = "go test -bench=`"$Benchmark`" -benchtime=`"$BenchTime`" -count=$Count -cpu=`"$CPU`" -benchmem -cpuprofile=`"$CpuProfile`" -memprofile=`"$MemProfile`" ./..."
Write-Host "Running: $benchmarkCommand"
Invoke-Expression $benchmarkCommand | Tee-Object -FilePath $ResultFile

# Generate JSON output for programmatic analysis
# 生成用于程序化分析的JSON输出
try {
    go run golang.org/x/perf/cmd/benchstat -json $ResultFile > $JsonFile
    Write-Host "JSON output generated: $JsonFile"
} catch {
    Write-Host "Failed to generate JSON output. Make sure golang.org/x/perf/cmd/benchstat is installed."
    Write-Host "Run: go get golang.org/x/perf/cmd/benchstat"
}

# Generate Markdown output for human-readable documentation
# 生成人类可读的Markdown文档
"# Benchmark Results" | Out-File -FilePath $MarkdownFile
"" | Out-File -FilePath $MarkdownFile -Append
"Date: $(Get-Date)" | Out-File -FilePath $MarkdownFile -Append
"" | Out-File -FilePath $MarkdownFile -Append
"## System Information" | Out-File -FilePath $MarkdownFile -Append
"" | Out-File -FilePath $MarkdownFile -Append
"- OS: Windows $(Get-CimInstance Win32_OperatingSystem | Select-Object -ExpandProperty Version)" | Out-File -FilePath $MarkdownFile -Append
"- CPU: $(Get-CimInstance Win32_Processor | Select-Object -ExpandProperty Name)" | Out-File -FilePath $MarkdownFile -Append
"- Memory: $([math]::Round((Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory / 1GB, 2)) GB" | Out-File -FilePath $MarkdownFile -Append
"- Go version: $(go version)" | Out-File -FilePath $MarkdownFile -Append
"" | Out-File -FilePath $MarkdownFile -Append
"## Results" | Out-File -FilePath $MarkdownFile -Append
"" | Out-File -FilePath $MarkdownFile -Append
"```" | Out-File -FilePath $MarkdownFile -Append
Get-Content $ResultFile | Out-File -FilePath $MarkdownFile -Append
"```" | Out-File -FilePath $MarkdownFile -Append

# Compare with previous results if requested
# 如果请求，则与先前的结果进行比较
if ($CompareWith -ne "") {
    "" | Out-File -FilePath $MarkdownFile -Append
    "## Comparison with Previous Results" | Out-File -FilePath $MarkdownFile -Append
    "" | Out-File -FilePath $MarkdownFile -Append
    "```" | Out-File -FilePath $MarkdownFile -Append
    try {
        $comparison = go run golang.org/x/perf/cmd/benchstat $CompareWith $ResultFile
        $comparison | Out-File -FilePath $MarkdownFile -Append
    } catch {
        "Failed to generate comparison. Make sure golang.org/x/perf/cmd/benchstat is installed." | Out-File -FilePath $MarkdownFile -Append
    }
    "```" | Out-File -FilePath $MarkdownFile -Append
}

# Print summary of output files
# 打印输出文件的摘要
Write-Host ""
Write-Host "Benchmark results written to:"
Write-Host "  $ResultFile"
Write-Host "  $JsonFile"
Write-Host "  $MarkdownFile"
Write-Host ""
Write-Host "Profiles written to:"
Write-Host "  $CpuProfile"
Write-Host "  $MemProfile"
Write-Host ""
Write-Host "To analyze CPU profile:"
Write-Host "  go tool pprof -http=:8080 $CpuProfile"
Write-Host ""
Write-Host "To analyze memory profile:"
Write-Host "  go tool pprof -http=:8081 $MemProfile" 