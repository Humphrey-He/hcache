# PowerShell 版本的 vegeta 压测脚本

# 检查 vegeta 是否已安装
if (-not (Get-Command vegeta -ErrorAction SilentlyContinue)) {
    Write-Error "错误: 未找到 vegeta 工具，请先安装。"
    Write-Host "安装指南: https://github.com/tsenart/vegeta#install"
    exit 1
}

# 默认参数
$rate = 100
$duration = "30s"
$targets = "vegeta.json"
$output = "results"
$format = "html"

# 解析命令行参数
param(
    [Alias('r')][int]$Rate = 100,
    [Alias('d')][string]$Duration = "30s",
    [Alias('t')][string]$TargetsFile = "vegeta.json",
    [Alias('o')][string]$OutputPrefix = "results",
    [Alias('f')][string]$Format = "html"
)

# 使用参数值
$rate = $Rate
$duration = $Duration
$targets = $TargetsFile
$output = $OutputPrefix
$format = $Format

# 创建时间戳
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$resultFile = "${output}_${rate}rps_${timestamp}"

Write-Host "启动压测..."
Write-Host "速率: $rate 请求/秒"
Write-Host "持续时间: $duration"
Write-Host "目标文件: $targets"

# 执行压测
$binResult = vegeta attack -targets="$targets" -rate="$rate" -duration="$duration"
$binResult | Out-File -FilePath "${resultFile}.bin" -Encoding ascii
$binResult | vegeta report

# 生成报告
switch ($format) {
    "html" {
        Get-Content "${resultFile}.bin" | vegeta plot > "${resultFile}.html"
        Write-Host "HTML 报告已生成: ${resultFile}.html"
    }
    "json" {
        Get-Content "${resultFile}.bin" | vegeta report -type=json > "${resultFile}.json"
        Write-Host "JSON 报告已生成: ${resultFile}.json"
    }
    "text" {
        Get-Content "${resultFile}.bin" | vegeta report > "${resultFile}.txt"
        Write-Host "文本报告已生成: ${resultFile}.txt"
    }
    default {
        Write-Error "未知格式: $format"
        exit 1
    }
}

# 生成直方图
Get-Content "${resultFile}.bin" | vegeta report -type="hist[0,1ms,5ms,10ms,25ms,50ms,100ms,250ms,500ms]" > "${resultFile}_hist.txt"
Write-Host "直方图报告已生成: ${resultFile}_hist.txt"

Write-Host "压测完成!" 