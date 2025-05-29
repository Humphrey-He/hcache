#!/bin/bash

# 确保已安装 vegeta
if ! command -v vegeta &> /dev/null; then
    echo "错误: 未找到 vegeta 工具，请先安装。"
    echo "安装指南: https://github.com/tsenart/vegeta#install"
    exit 1
fi

# 默认参数
RATE=100
DURATION=30s
TARGETS="vegeta.json"
OUTPUT="results"
FORMAT="html"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    -r|--rate)
      RATE="$2"
      shift
      shift
      ;;
    -d|--duration)
      DURATION="$2"
      shift
      shift
      ;;
    -t|--targets)
      TARGETS="$2"
      shift
      shift
      ;;
    -o|--output)
      OUTPUT="$2"
      shift
      shift
      ;;
    -f|--format)
      FORMAT="$2"
      shift
      shift
      ;;
    -h|--help)
      echo "用法: $0 [选项]"
      echo "选项:"
      echo "  -r, --rate RATE       每秒请求数 (默认: 100)"
      echo "  -d, --duration DUR    测试持续时间 (默认: 30s)"
      echo "  -t, --targets FILE    目标文件 (默认: vegeta.json)"
      echo "  -o, --output PREFIX   输出文件前缀 (默认: results)"
      echo "  -f, --format FORMAT   输出格式 (html, json, text, 默认: html)"
      echo "  -h, --help            显示此帮助信息"
      exit 0
      ;;
    *)
      echo "未知选项: $key"
      exit 1
      ;;
  esac
done

# 创建时间戳
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
RESULT_FILE="${OUTPUT}_${RATE}rps_${TIMESTAMP}"

echo "启动压测..."
echo "速率: $RATE 请求/秒"
echo "持续时间: $DURATION"
echo "目标文件: $TARGETS"

# 执行压测
vegeta attack -targets="$TARGETS" -rate="$RATE" -duration="$DURATION" | tee "${RESULT_FILE}.bin" | vegeta report

# 生成报告
case $FORMAT in
  html)
    cat "${RESULT_FILE}.bin" | vegeta plot > "${RESULT_FILE}.html"
    echo "HTML 报告已生成: ${RESULT_FILE}.html"
    ;;
  json)
    cat "${RESULT_FILE}.bin" | vegeta report -type=json > "${RESULT_FILE}.json"
    echo "JSON 报告已生成: ${RESULT_FILE}.json"
    ;;
  text)
    cat "${RESULT_FILE}.bin" | vegeta report > "${RESULT_FILE}.txt"
    echo "文本报告已生成: ${RESULT_FILE}.txt"
    ;;
  *)
    echo "未知格式: $FORMAT"
    exit 1
    ;;
esac

# 生成直方图
cat "${RESULT_FILE}.bin" | vegeta report -type="hist[0,1ms,5ms,10ms,25ms,50ms,100ms,250ms,500ms]" > "${RESULT_FILE}_hist.txt"
echo "直方图报告已生成: ${RESULT_FILE}_hist.txt"

echo "压测完成!" 