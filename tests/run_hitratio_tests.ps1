# Run HitRatio Tests and Collect Results
# 运行命中率测试并收集结果

<#
.DESCRIPTION
    This script executes specialized hit ratio tests for HCache and collects the results.
    It tests how different cache eviction policies perform under various access patterns.
    
    此脚本执行 HCache 的专业命中率测试并收集结果。
    它测试不同的缓存淘汰策略在各种访问模式下的表现。

.TEST TARGETS
    - Contention Resistance: Tests how cache performs when multiple access patterns compete for cache space
      竞争抵抗：测试当多种访问模式竞争缓存空间时的性能
    
    - Search Pattern: Simulates search engine query patterns with few hot items and many rare items
      搜索模式：模拟搜索引擎查询模式，包含少量热门项目和大量罕见项目
    
    - Database Pattern: Simulates database access patterns including record access and index lookups
      数据库模式：模拟数据库访问模式，包括记录访问和索引查找
    
    - Looping Pattern: Simulates repeated access to the same dataset in a cyclical pattern
      循环模式：模拟以循环模式重复访问相同数据集
    
    - CODASYL Pattern: Simulates network database patterns where data is accessed in a graph structure
      CODASYL模式：模拟网络数据库模式，数据在图结构中被访问

.CACHE POLICIES TESTED
    - LRU (Least Recently Used): Evicts least recently accessed items first
      LRU（最近最少使用）：首先淘汰最近最少访问的项目
    
    - LFU (Least Frequently Used): Evicts least frequently accessed items first
      LFU（最不经常使用）：首先淘汰访问频率最低的项目
    
    - FIFO (First In First Out): Evicts oldest items first, regardless of access frequency
      FIFO（先进先出）：首先淘汰最早的项目，不考虑访问频率
    
    - Random: Randomly selects items to evict, serving as a baseline
      随机：随机选择要淘汰的项目，作为基准线

.CACHE SIZES TESTED
    Tests are performed with two cache sizes to evaluate scaling behavior:
    - Small cache: 1000 items
    - Large cache: 10000 items
    
    测试使用两种缓存大小来评估扩展行为：
    - 小缓存：1000个项目
    - 大缓存：10000个项目
#>

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
    
    # Define the policies and cache sizes being tested
    # 定义被测试的策略和缓存大小
    $policies = @("lru", "lfu", "fifo", "random")
    $sizes = @(1000, 10000)
    
    # Map hit ratio data to policies and sizes
    # 将命中率数据映射到策略和大小
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

# For each test pattern, create a summary section in the report
# 为每个测试模式在报告中创建摘要部分
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