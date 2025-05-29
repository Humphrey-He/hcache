# HCache Performance Analysis Toolkit

This toolkit provides comprehensive analysis capabilities for HCache performance testing results. It processes data from benchmark tests, hit ratio tests, concurrency tests, and performance profiles to generate insightful visualizations and reports.

## Features

- **Benchmark Analysis**: Processes Go benchmark results to analyze latency, memory allocations, and throughput
- **Hit Ratio Analysis**: Analyzes cache hit ratio test results across different eviction policies and access patterns
- **Concurrency Analysis**: Evaluates performance under different concurrency levels
- **pprof Analysis**: Generates flamegraphs and identifies hotspots from Go pprof profiles
- **Comprehensive Analysis**: Combines all analyses into a single comprehensive report

## Requirements

- Python 3.8+
- Required Python packages (install using `pip install -r requirements.txt`):
  - pandas
  - numpy
  - matplotlib
  - seaborn
  - plotly
  - scikit-learn
  - kaleido
  - nbformat
  - jupyter
  - py-cpuinfo
  - openpyxl
- Go toolchain (for pprof analysis)

## Directory Structure

```
tests/analysis/
├── analyze_all.py            # Main script for comprehensive analysis
├── benchmark_analyzer.py     # Benchmark analysis module
├── hitratio_analyzer.py      # Hit ratio analysis module
├── concurrency_analyzer.py   # Concurrency analysis module
├── pprof_analyzer.py         # pprof analysis module
├── requirements.txt          # Python dependencies
└── README.md                 # This file
```

## Usage

### Running the Comprehensive Analysis

To run a comprehensive analysis of all test results:

```bash
python analyze_all.py --base-dir /path/to/tests --output ./analysis_results
```

Options:
- `--base-dir`, `-b`: Base directory containing test results (default: `..`)
- `--output`, `-o`: Directory to save analysis results (default: `./output`)
- `--skip-benchmark`: Skip benchmark analysis
- `--skip-hitratio`: Skip hit ratio analysis
- `--skip-concurrency`: Skip concurrency analysis
- `--skip-pprof`: Skip pprof analysis

### Running Individual Analyzers

You can also run each analyzer individually:

#### Benchmark Analysis

```bash
python benchmark_analyzer.py --input /path/to/benchmark/results --output ./benchmark_analysis
```

#### Hit Ratio Analysis

```bash
python hitratio_analyzer.py --input /path/to/hitratio/results --output ./hitratio_analysis
```

#### Concurrency Analysis

```bash
python concurrency_analyzer.py --input /path/to/concurrency/results --output ./concurrency_analysis
```

#### pprof Analysis

```bash
python pprof_analyzer.py --input /path/to/pprof/profiles --output ./pprof_analysis
```

## Output

The analysis toolkit generates the following outputs:

- **CSV Files**: Raw data and statistics in CSV format
- **Images**: Static plots and charts in PNG format
- **HTML Files**: Interactive visualizations using Plotly
- **Markdown Reports**: Detailed analysis reports in Markdown format
- **HTML Reports**: Interactive HTML versions of the reports
- **Excel Summary**: Key metrics summarized in Excel format
- **Flamegraphs**: SVG flamegraphs for CPU and memory profiles (pprof analysis only)

## Example Workflow

1. Run your HCache tests to generate benchmark, hit ratio, concurrency, and pprof data
2. Run the comprehensive analysis:
   ```bash
   python analyze_all.py --base-dir /path/to/tests --output ./analysis_results
   ```
3. Review the comprehensive report in `analysis_results/summary/comprehensive_report_YYYYMMDD.md`
4. Explore detailed reports and visualizations in the respective subdirectories

## Extending the Toolkit

To add new analysis capabilities:

1. Create a new analyzer module following the pattern of existing analyzers
2. Implement the core analysis functions
3. Update `analyze_all.py` to incorporate the new analyzer

## License

This toolkit is part of the HCache project and is subject to the same license terms.

## Contributors

- [Your Name/Team]

## Acknowledgments

- The Go team for pprof
- The Python data science community for pandas, matplotlib, and other tools

## Hit Ratio Analysis

The `hitratio_visualizer.py` script provides visualization tools for hit ratio test results. It creates various charts to help understand cache performance under different access patterns:

### Specialized Access Patterns

The visualizer supports analysis of the following specialized access patterns:

1. **Contention Resistance**: How the cache performs when multiple access patterns compete for the same cache space.
2. **Search Pattern**: Simulates search engine query patterns with few popular terms and many rare terms.
3. **Database Pattern**: Models database access patterns including record access and index lookups.
4. **Looping Pattern**: Represents repeated access to the same set of data in loops.
5. **CODASYL Pattern**: Simulates network database access where data is traversed in a graph structure.

### Visualization Types

- **Bar Charts**: Compare hit ratios by policy and cache size for each test pattern.
- **Policy Comparison Charts**: Compare different policies across all test patterns.
- **Heatmaps**: Show hit ratios across all test patterns and policies.
- **Radar Charts**: Visualize policy performance across different test patterns.

## Usage

### Requirements

Install the required Python packages:

```bash
pip install -r requirements.txt
```

### Running the Hit Ratio Visualizer

```bash
python hitratio_visualizer.py
```

By default, the script looks for CSV files in the `results/hitratio` directory and saves visualizations to `results/hitratio/visualizations`.

### Workflow

1. Run hit ratio tests using the PowerShell script:
   ```powershell
   ./tests/run_hitratio_tests.ps1
   ```

2. Generate visualizations:
   ```bash
   cd tests/analysis
   python hitratio_visualizer.py
   ```

3. View the results in the `results/hitratio/visualizations` directory.

## Output

The visualizer produces the following outputs:

- PNG image files for each visualization type
- CSV files with processed data
- A summary report in Markdown format

## Contributing

When adding new analysis tools, please follow these guidelines:

1. Use bilingual comments (English and Chinese) for better accessibility.
2. Follow Python docstring conventions for function and class documentation.
3. Provide clear error handling and user feedback.
4. Include example usage in the tool's documentation. 