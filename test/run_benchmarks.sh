#!/bin/bash

# This script runs benchmarks and generates reports for the HCache library.
# It provides various options to customize benchmark execution and output.
#
# 此脚本运行HCache库的基准测试并生成报告。
# 它提供了各种选项来自定义基准测试执行和输出。

# Default values for benchmark parameters
# 基准测试参数的默认值
BENCHMARK="."
BENCHTIME="3s"
COUNT=3
CPU="1,4,8"
OUTPUT_DIR="benchmark_results"
COMPARE_WITH=""

# Parse command line arguments
# 解析命令行参数
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -b|--benchmark)
      # Pattern to match benchmark names
      # 匹配基准测试名称的模式
      BENCHMARK="$2"
      shift
      shift
      ;;
    -t|--time)
      # Duration to run each benchmark
      # 运行每个基准测试的持续时间
      BENCHTIME="$2"
      shift
      shift
      ;;
    -c|--count)
      # Number of times to run each benchmark
      # 运行每个基准测试的次数
      COUNT="$2"
      shift
      shift
      ;;
    --cpu)
      # GOMAXPROCS values to test with
      # 要测试的GOMAXPROCS值
      CPU="$2"
      shift
      shift
      ;;
    -o|--output)
      # Directory to store results
      # 存储结果的目录
      OUTPUT_DIR="$2"
      shift
      shift
      ;;
    --compare)
      # Previous benchmark results to compare against
      # 要比较的先前基准测试结果
      COMPARE_WITH="$2"
      shift
      shift
      ;;
    -h|--help)
      # Display help information
      # 显示帮助信息
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  -b, --benchmark PATTERN   Benchmark pattern (default: '.')"
      echo "  -t, --time DURATION       Benchmark duration (default: '3s')"
      echo "  -c, --count N             Run each benchmark N times (default: 3)"
      echo "  --cpu LIST                Run with different GOMAXPROCS values (default: '1,4,8')"
      echo "  -o, --output DIR          Output directory (default: 'benchmark_results')"
      echo "  --compare FILE            Compare with previous benchmark results"
      echo "  -h, --help                Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Create output directory if it doesn't exist
# 如果输出目录不存在，则创建它
mkdir -p "$OUTPUT_DIR"

# Get current date and time for unique filenames
# 获取当前日期和时间以生成唯一的文件名
DATE=$(date +"%Y%m%d_%H%M%S")
RESULT_FILE="$OUTPUT_DIR/benchmark_$DATE.txt"
JSON_FILE="$OUTPUT_DIR/benchmark_$DATE.json"
MARKDOWN_FILE="$OUTPUT_DIR/benchmark_$DATE.md"
CPU_PROFILE="$OUTPUT_DIR/cpu_$DATE.prof"
MEM_PROFILE="$OUTPUT_DIR/mem_$DATE.prof"

# Print benchmark information before starting
# 在开始前打印基准测试信息
echo "Running benchmarks:"
echo "  Pattern: $BENCHMARK"
echo "  Duration: $BENCHTIME"
echo "  Count: $COUNT"
echo "  CPU: $CPU"
echo "  Output: $RESULT_FILE"
echo

# Run benchmarks with the specified parameters
# 使用指定的参数运行基准测试
go test -bench="$BENCHMARK" -benchtime="$BENCHTIME" -count="$COUNT" -cpu="$CPU" \
  -benchmem -cpuprofile="$CPU_PROFILE" -memprofile="$MEM_PROFILE" \
  ./... | tee "$RESULT_FILE"

# Generate JSON output for programmatic analysis
# 生成用于程序化分析的JSON输出
go run golang.org/x/perf/cmd/benchstat -json "$RESULT_FILE" > "$JSON_FILE"

# Generate Markdown output for human-readable documentation
# 生成人类可读的Markdown文档
echo "# Benchmark Results" > "$MARKDOWN_FILE"
echo >> "$MARKDOWN_FILE"
echo "Date: $(date)" >> "$MARKDOWN_FILE"
echo >> "$MARKDOWN_FILE"
echo "## System Information" >> "$MARKDOWN_FILE"
echo >> "$MARKDOWN_FILE"
echo "- OS: $(uname -s)" >> "$MARKDOWN_FILE"
echo "- CPU: $(grep -m 1 'model name' /proc/cpuinfo | cut -d ':' -f 2 | xargs)" >> "$MARKDOWN_FILE"
echo "- Memory: $(free -h | grep Mem | awk '{print $2}')" >> "$MARKDOWN_FILE"
echo "- Go version: $(go version)" >> "$MARKDOWN_FILE"
echo >> "$MARKDOWN_FILE"
echo "## Results" >> "$MARKDOWN_FILE"
echo >> "$MARKDOWN_FILE"
echo '```' >> "$MARKDOWN_FILE"
cat "$RESULT_FILE" >> "$MARKDOWN_FILE"
echo '```' >> "$MARKDOWN_FILE"

# Compare with previous results if requested
# 如果请求，则与先前的结果进行比较
if [ -n "$COMPARE_WITH" ]; then
  echo >> "$MARKDOWN_FILE"
  echo "## Comparison with Previous Results" >> "$MARKDOWN_FILE"
  echo >> "$MARKDOWN_FILE"
  echo '```' >> "$MARKDOWN_FILE"
  go run golang.org/x/perf/cmd/benchstat "$COMPARE_WITH" "$RESULT_FILE" >> "$MARKDOWN_FILE"
  echo '```' >> "$MARKDOWN_FILE"
fi

# Print summary of output files
# 打印输出文件的摘要
echo
echo "Benchmark results written to:"
echo "  $RESULT_FILE"
echo "  $JSON_FILE"
echo "  $MARKDOWN_FILE"
echo
echo "Profiles written to:"
echo "  $CPU_PROFILE"
echo "  $MEM_PROFILE"
echo
echo "To analyze CPU profile:"
echo "  go tool pprof -http=:8080 $CPU_PROFILE"
echo
echo "To analyze memory profile:"
echo "  go tool pprof -http=:8081 $MEM_PROFILE" 