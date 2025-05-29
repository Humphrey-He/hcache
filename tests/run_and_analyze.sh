#!/bin/bash
# Script to run all HCache tests and analyze the results

# Set the base directory to the script's location
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$BASE_DIR/results"
ANALYSIS_DIR="$RESULTS_DIR/analysis"

# Create results directories
mkdir -p "$RESULTS_DIR/benchmark"
mkdir -p "$RESULTS_DIR/hitratio"
mkdir -p "$RESULTS_DIR/concurrency"
mkdir -p "$RESULTS_DIR/pprof"

# Function to display section headers
section() {
    echo ""
    echo "====================================================="
    echo "  $1"
    echo "====================================================="
    echo ""
}

# Check if Python and required packages are installed
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
run_benchmarks() {
    section "Running benchmark tests"
    
    cd "$BASE_DIR/benchmark"
    
    # Enable CPU profiling
    export CPUPROFILE="$RESULTS_DIR/pprof/benchmark_cpu_$(date +%Y%m%d).pprof"
    export MEMPROFILE="$RESULTS_DIR/pprof/benchmark_mem_$(date +%Y%m%d).pprof"
    
    echo "Running benchmark tests..."
    go test -bench=. -benchmem -count=5 | tee "$RESULTS_DIR/benchmark/benchmark_$(date +%Y%m%d).txt"
    
    echo "Benchmark tests completed."
}

# Run hit ratio tests
run_hitratio_tests() {
    section "Running hit ratio tests"
    
    cd "$BASE_DIR/hitratio"
    
    echo "Running hit ratio tests..."
    go test -v | tee "$RESULTS_DIR/hitratio/hitratio_$(date +%Y%m%d).txt"
    
    echo "Hit ratio tests completed."
}

# Run concurrency tests
run_concurrency_tests() {
    section "Running concurrency tests"
    
    cd "$BASE_DIR/concurrency"
    
    echo "Starting mock server..."
    cd mock_server
    go run main.go &
    SERVER_PID=$!
    
    # Wait for server to start
    sleep 2
    
    echo "Running vegeta load tests..."
    cd ../vegeta
    
    # Run with different concurrency levels
    for CONCURRENCY in 1 10 50 100 200; do
        echo "Running with concurrency $CONCURRENCY..."
        
        # Adjust rate based on concurrency
        RATE=$((CONCURRENCY * 10))
        
        # Run for 30 seconds
        echo "vegeta attack -rate=$RATE -duration=30s -targets=targets.txt | vegeta report -type=json > $RESULTS_DIR/concurrency/vegeta_c${CONCURRENCY}_r${RATE}_$(date +%Y%m%d).json"
        vegeta attack -rate=$RATE -duration=30s -targets=targets.txt | vegeta report -type=json > "$RESULTS_DIR/concurrency/vegeta_c${CONCURRENCY}_r${RATE}_$(date +%Y%m%d).json"
    done
    
    # Stop the mock server
    kill $SERVER_PID
    
    echo "Concurrency tests completed."
}

# Run pprof analysis
run_pprof_analysis() {
    section "Running pprof analysis"
    
    # Check if pprof profiles exist
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
run_analysis() {
    section "Running analysis tools"
    
    cd "$BASE_DIR/analysis"
    
    echo "Running comprehensive analysis..."
    python3 analyze_all.py --base-dir "$RESULTS_DIR" --output "$ANALYSIS_DIR"
    
    echo "Analysis completed. Results are available in $ANALYSIS_DIR"
}

# Main execution
main() {
    section "HCache Test and Analysis Suite"
    echo "Starting tests and analysis at $(date)"
    
    # Check dependencies
    check_python_deps
    
    # Ask user which tests to run
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
    run_analysis
    
    section "Test and Analysis Completed"
    echo "All tests and analysis completed at $(date)"
    echo "Results are available in $RESULTS_DIR"
    echo "Analysis results are available in $ANALYSIS_DIR"
}

# Run the main function
main 