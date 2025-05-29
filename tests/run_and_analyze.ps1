# PowerShell script to run all HCache tests and analyze the results

# Set the base directory to the script's location
$BASE_DIR = $PSScriptRoot
$RESULTS_DIR = Join-Path $BASE_DIR "results"
$ANALYSIS_DIR = Join-Path $RESULTS_DIR "analysis"

# Create results directories
New-Item -Path (Join-Path $RESULTS_DIR "benchmark") -ItemType Directory -Force | Out-Null
New-Item -Path (Join-Path $RESULTS_DIR "hitratio") -ItemType Directory -Force | Out-Null
New-Item -Path (Join-Path $RESULTS_DIR "concurrency") -ItemType Directory -Force | Out-Null
New-Item -Path (Join-Path $RESULTS_DIR "pprof") -ItemType Directory -Force | Out-Null

# Function to display section headers
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
function Start-BenchmarkTests {
    Show-Section "Running benchmark tests"
    
    Push-Location (Join-Path $BASE_DIR "benchmark")
    
    # Enable CPU profiling
    $env:CPUPROFILE = Join-Path $RESULTS_DIR "pprof\benchmark_cpu_$(Get-Date -Format 'yyyyMMdd').pprof"
    $env:MEMPROFILE = Join-Path $RESULTS_DIR "pprof\benchmark_mem_$(Get-Date -Format 'yyyyMMdd').pprof"
    
    Write-Host "Running benchmark tests..."
    $benchmarkOutput = Join-Path $RESULTS_DIR "benchmark\benchmark_$(Get-Date -Format 'yyyyMMdd').txt"
    go test -bench=. -benchmem -count=5 | Tee-Object -FilePath $benchmarkOutput
    
    Write-Host "Benchmark tests completed."
    
    Pop-Location
}

# Run hit ratio tests
function Start-HitRatioTests {
    Show-Section "Running hit ratio tests"
    
    Push-Location (Join-Path $BASE_DIR "hitratio")
    
    Write-Host "Running hit ratio tests..."
    $hitratioOutput = Join-Path $RESULTS_DIR "hitratio\hitratio_$(Get-Date -Format 'yyyyMMdd').txt"
    go test -v | Tee-Object -FilePath $hitratioOutput
    
    Write-Host "Hit ratio tests completed."
    
    Pop-Location
}

# Run concurrency tests
function Start-ConcurrencyTests {
    Show-Section "Running concurrency tests"
    
    Push-Location (Join-Path $BASE_DIR "concurrency")
    
    Write-Host "Starting mock server..."
    Push-Location "mock_server"
    $serverJob = Start-Job -ScriptBlock { 
        Set-Location $using:PWD
        go run main.go 
    }
    
    # Wait for server to start
    Start-Sleep -Seconds 2
    
    Write-Host "Running vegeta load tests..."
    Push-Location "..\vegeta"
    
    # Run with different concurrency levels
    foreach ($concurrency in @(1, 10, 50, 100, 200)) {
        Write-Host "Running with concurrency $concurrency..."
        
        # Adjust rate based on concurrency
        $rate = $concurrency * 10
        
        # Run for 30 seconds
        $outputFile = Join-Path $RESULTS_DIR "concurrency\vegeta_c${concurrency}_r${rate}_$(Get-Date -Format 'yyyyMMdd').json"
        Write-Host "vegeta attack -rate=$rate -duration=30s -targets=targets.txt | vegeta report -type=json > $outputFile"
        
        # Check if vegeta is installed
        try {
            vegeta attack -rate=$rate -duration=30s -targets=targets.txt | vegeta report -type=json | Out-File -FilePath $outputFile
        }
        catch {
            Write-Host "Vegeta is not installed. Please install vegeta to run concurrency tests." -ForegroundColor Yellow
            break
        }
    }
    
    # Stop the mock server
    Stop-Job -Job $serverJob
    Remove-Job -Job $serverJob -Force
    
    Write-Host "Concurrency tests completed."
    
    Pop-Location
    Pop-Location
}

# Run pprof analysis
function Start-PprofAnalysis {
    Show-Section "Running pprof analysis"
    
    # Check if pprof profiles exist
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
function Start-Analysis {
    Show-Section "Running analysis tools"
    
    Push-Location (Join-Path $BASE_DIR "analysis")
    
    Write-Host "Running comprehensive analysis..."
    python analyze_all.py --base-dir $RESULTS_DIR --output $ANALYSIS_DIR
    
    Write-Host "Analysis completed. Results are available in $ANALYSIS_DIR"
    
    Pop-Location
}

# Main execution
function Start-Main {
    Show-Section "HCache Test and Analysis Suite"
    Write-Host "Starting tests and analysis at $(Get-Date)"
    
    # Check dependencies
    Test-PythonDeps
    
    # Ask user which tests to run
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
    Start-Analysis
    
    Show-Section "Test and Analysis Completed"
    Write-Host "All tests and analysis completed at $(Get-Date)"
    Write-Host "Results are available in $RESULTS_DIR"
    Write-Host "Analysis results are available in $ANALYSIS_DIR"
}

# Run the main function
Start-Main 