# Run HitRatio Tests and Collect Results
# 运行命中率测试并收集结果

# Define test patterns to run
# 定义要运行的测试模式
$testPatterns = @(
    "TestContentionResistance",
    "TestSearchPattern",
    "TestDatabasePattern",
    "TestLoopingPattern",
    "TestCODASYLPattern"
)

# Create results directory if it doesn't exist
# 如果结果目录不存在，则创建它
$resultsDir = "results/hitratio"
if (-not (Test-Path $resultsDir)) {
    New-Item -ItemType Directory -Path $resultsDir -Force | Out-Null
}

# Run each test and save results
# 运行每个测试并保存结果
foreach ($pattern in $testPatterns) {
    Write-Host "Running $pattern tests..." -ForegroundColor Green
    
    # Run the test and capture output
    # 运行测试并捕获输出
    $output = & go test -v "./tests/hitratio" -run $pattern -count=1 2>&1
    
    # Save output to file
    # 将输出保存到文件
    $output | Out-File -FilePath "$resultsDir/$pattern.txt"
    
    # Extract hit ratio data
    # 提取命中率数据
    $hitRatioData = $output | Select-String -Pattern "命中率: ([0-9.]+)%" | ForEach-Object { $_.Matches.Groups[1].Value }
    
    # Create CSV file with hit ratio data
    # 使用命中率数据创建CSV文件
    $csvPath = "$resultsDir/$pattern.csv"
    "Policy,CacheSize,HitRatio" | Out-File -FilePath $csvPath
    
    $policies = @("lru", "lfu", "fifo", "random")
    $sizes = @(1000, 10000)
    
    $i = 0
    foreach ($size in $sizes) {
        foreach ($policy in $policies) {
            if ($i -lt $hitRatioData.Count) {
                "$policy,$size,$($hitRatioData[$i])" | Out-File -FilePath $csvPath -Append
                $i++
            }
        }
    }
    
    Write-Host "Results saved to $resultsDir/$pattern.txt and $csvPath" -ForegroundColor Cyan
}

# Generate summary report
# 生成摘要报告
$summaryPath = "$resultsDir/summary.md"

"# Hit Ratio Test Results Summary" | Out-File -FilePath $summaryPath
"" | Out-File -FilePath $summaryPath -Append
"Generated on: $(Get-Date)" | Out-File -FilePath $summaryPath -Append
"" | Out-File -FilePath $summaryPath -Append

foreach ($pattern in $testPatterns) {
    "## $pattern" | Out-File -FilePath $summaryPath -Append
    "" | Out-File -FilePath $summaryPath -Append
    
    $csvPath = "$resultsDir/$pattern.csv"
    if (Test-Path $csvPath) {
        $data = Import-Csv -Path $csvPath
        
        "| Policy | Cache Size | Hit Ratio (%) |" | Out-File -FilePath $summaryPath -Append
        "| ------ | ---------- | ------------- |" | Out-File -FilePath $summaryPath -Append
        
        foreach ($row in $data) {
            "| $($row.Policy) | $($row.CacheSize) | $($row.HitRatio) |" | Out-File -FilePath $summaryPath -Append
        }
        
        "" | Out-File -FilePath $summaryPath -Append
    } else {
        "No data available for this test pattern." | Out-File -FilePath $summaryPath -Append
        "" | Out-File -FilePath $summaryPath -Append
    }
}

Write-Host "Summary report generated at $summaryPath" -ForegroundColor Green
Write-Host "Tests completed!" -ForegroundColor Green 