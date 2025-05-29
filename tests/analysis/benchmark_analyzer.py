#!/usr/bin/env python3
"""
HCache Benchmark Analysis Tool

This script processes benchmark test results from Go's testing package and generates
comprehensive analysis and visualizations.
"""

import os
import re
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
import plotly.express as px
import plotly.graph_objects as go
from plotly.subplots import make_subplots
from datetime import datetime
import json
import argparse

# Set style for matplotlib
plt.style.use('ggplot')
sns.set_theme(style="whitegrid")

class BenchmarkAnalyzer:
    """Analyzes Go benchmark results and generates visualizations."""
    
    def __init__(self, input_dir, output_dir):
        """
        Initialize the analyzer with input and output directories.
        
        Args:
            input_dir: Directory containing benchmark result files
            output_dir: Directory to save analysis results
        """
        self.input_dir = input_dir
        self.output_dir = output_dir
        self.results = None
        self.comparison_results = {}
        
        # Create output directory if it doesn't exist
        os.makedirs(output_dir, exist_ok=True)
        
        # Create subdirectories for different output types
        self.img_dir = os.path.join(output_dir, 'images')
        self.csv_dir = os.path.join(output_dir, 'csv')
        self.html_dir = os.path.join(output_dir, 'html')
        self.report_dir = os.path.join(output_dir, 'reports')
        
        for directory in [self.img_dir, self.csv_dir, self.html_dir, self.report_dir]:
            os.makedirs(directory, exist_ok=True)
    
    def parse_benchmark_file(self, filename):
        """
        Parse a Go benchmark result file into a pandas DataFrame.
        
        Args:
            filename: Path to the benchmark result file
            
        Returns:
            DataFrame containing parsed benchmark results
        """
        with open(filename, 'r') as f:
            content = f.read()
        
        # Regular expression to match benchmark lines
        pattern = r'^(Benchmark\w+)(?:-(\d+))?\s+(\d+)\s+(\d+(?:\.\d+)?) ns/op(?:\s+(\d+) B/op)?(?:\s+(\d+) allocs/op)?$'
        
        results = []
        for line in content.split('\n'):
            match = re.match(pattern, line)
            if match:
                name, procs, iterations, ns_per_op, bytes_per_op, allocs_per_op = match.groups()
                
                # Extract test parameters from name
                params = {}
                name_parts = name.split('/')
                base_name = name_parts[0]
                
                for part in name_parts[1:]:
                    if '=' in part:
                        key, value = part.split('=', 1)
                        params[key] = value
                
                result = {
                    'name': base_name,
                    'full_name': name,
                    'procs': int(procs) if procs else 1,
                    'iterations': int(iterations),
                    'ns_per_op': float(ns_per_op),
                    'bytes_per_op': int(bytes_per_op) if bytes_per_op else 0,
                    'allocs_per_op': int(allocs_per_op) if allocs_per_op else 0
                }
                
                # Add extracted parameters
                result.update(params)
                
                results.append(result)
        
        return pd.DataFrame(results)
    
    def load_benchmark_results(self, pattern=None):
        """
        Load all benchmark results from the input directory.
        
        Args:
            pattern: Optional regex pattern to filter files
            
        Returns:
            DataFrame containing all benchmark results
        """
        all_results = []
        
        for filename in os.listdir(self.input_dir):
            if not filename.endswith('.txt'):
                continue
            
            if pattern and not re.search(pattern, filename):
                continue
            
            filepath = os.path.join(self.input_dir, filename)
            
            # Extract metadata from filename
            match = re.search(r'(\w+)_(\d{8})\.txt', filename)
            if match:
                test_type, date_str = match.groups()
            else:
                test_type = 'unknown'
                date_str = '00000000'
            
            df = self.parse_benchmark_file(filepath)
            
            # Add metadata columns
            df['test_type'] = test_type
            df['date'] = pd.to_datetime(date_str, format='%Y%m%d')
            df['source_file'] = filename
            
            all_results.append(df)
        
        if not all_results:
            raise ValueError(f"No benchmark results found in {self.input_dir}")
        
        self.results = pd.concat(all_results, ignore_index=True)
        return self.results
    
    def load_comparison_data(self, other_cache_dir):
        """
        Load benchmark results for other cache libraries for comparison.
        
        Args:
            other_cache_dir: Directory containing benchmark results for other caches
            
        Returns:
            Dictionary of DataFrames with cache library names as keys
        """
        for subdir in os.listdir(other_cache_dir):
            cache_dir = os.path.join(other_cache_dir, subdir)
            if os.path.isdir(cache_dir):
                try:
                    all_results = []
                    for filename in os.listdir(cache_dir):
                        if filename.endswith('.txt'):
                            filepath = os.path.join(cache_dir, filename)
                            df = self.parse_benchmark_file(filepath)
                            
                            # Add metadata
                            match = re.search(r'(\w+)_(\d{8})\.txt', filename)
                            if match:
                                test_type, date_str = match.groups()
                            else:
                                test_type = 'unknown'
                                date_str = '00000000'
                            
                            df['test_type'] = test_type
                            df['date'] = pd.to_datetime(date_str, format='%Y%m%d')
                            df['source_file'] = filename
                            df['cache_lib'] = subdir
                            
                            all_results.append(df)
                    
                    if all_results:
                        self.comparison_results[subdir] = pd.concat(all_results, ignore_index=True)
                except Exception as e:
                    print(f"Error loading comparison data for {subdir}: {e}")
        
        return self.comparison_results
    
    def generate_descriptive_stats(self):
        """
        Generate descriptive statistics for benchmark results.
        
        Returns:
            DataFrame containing descriptive statistics
        """
        if self.results is None:
            raise ValueError("No benchmark results loaded")
        
        # Group by test name and calculate stats
        stats = self.results.groupby(['name', 'ValueSize']).agg({
            'ns_per_op': ['mean', 'median', 'std', 'min', 'max'],
            'bytes_per_op': ['mean', 'median'],
            'allocs_per_op': ['mean', 'median']
        }).reset_index()
        
        # Calculate percentiles
        percentiles = self.results.groupby(['name', 'ValueSize']).agg({
            'ns_per_op': lambda x: np.percentile(x, [50, 90, 95, 99])
        }).reset_index()
        
        # Rename percentile columns
        percentiles_df = pd.DataFrame({
            'name': percentiles['name'],
            'ValueSize': percentiles['ValueSize'],
            'p50': [p[0] for p in percentiles['ns_per_op']],
            'p90': [p[1] for p in percentiles['ns_per_op']],
            'p95': [p[2] for p in percentiles['ns_per_op']],
            'p99': [p[3] for p in percentiles['ns_per_op']]
        })
        
        # Save stats to CSV
        stats_file = os.path.join(self.csv_dir, 'descriptive_stats.csv')
        stats.to_csv(stats_file, index=False)
        
        percentiles_file = os.path.join(self.csv_dir, 'percentiles.csv')
        percentiles_df.to_csv(percentiles_file, index=False)
        
        return stats, percentiles_df
    
    def plot_performance_comparison(self):
        """
        Generate performance comparison plots.
        
        Returns:
            List of paths to generated plot files
        """
        if self.results is None:
            raise ValueError("No benchmark results loaded")
        
        plot_files = []
        
        # Group data by benchmark type
        benchmark_types = self.results['name'].unique()
        
        for benchmark_type in benchmark_types:
            # Filter data for this benchmark type
            benchmark_data = self.results[self.results['name'] == benchmark_type]
            
            if 'ValueSize' in benchmark_data.columns:
                # Plot ns/op by value size
                plt.figure(figsize=(10, 6))
                sns.barplot(x='ValueSize', y='ns_per_op', data=benchmark_data)
                plt.title(f'{benchmark_type} - Performance by Value Size')
                plt.xlabel('Value Size (bytes)')
                plt.ylabel('Time per Operation (ns)')
                plt.yscale('log')
                plt.grid(True, alpha=0.3)
                
                # Save plot
                plot_file = os.path.join(self.img_dir, f'{benchmark_type}_value_size.png')
                plt.savefig(plot_file, dpi=300, bbox_inches='tight')
                plt.close()
                plot_files.append(plot_file)
                
                # Interactive plot with Plotly
                fig = px.bar(
                    benchmark_data, 
                    x='ValueSize', 
                    y='ns_per_op',
                    title=f'{benchmark_type} - Performance by Value Size',
                    labels={'ValueSize': 'Value Size (bytes)', 'ns_per_op': 'Time per Operation (ns)'},
                    log_y=True
                )
                
                html_file = os.path.join(self.html_dir, f'{benchmark_type}_value_size.html')
                fig.write_html(html_file)
                plot_files.append(html_file)
            
            # Plot memory allocations
            if 'allocs_per_op' in benchmark_data.columns and 'ValueSize' in benchmark_data.columns:
                plt.figure(figsize=(10, 6))
                sns.barplot(x='ValueSize', y='allocs_per_op', data=benchmark_data)
                plt.title(f'{benchmark_type} - Memory Allocations by Value Size')
                plt.xlabel('Value Size (bytes)')
                plt.ylabel('Allocations per Operation')
                plt.grid(True, alpha=0.3)
                
                # Save plot
                plot_file = os.path.join(self.img_dir, f'{benchmark_type}_allocs.png')
                plt.savefig(plot_file, dpi=300, bbox_inches='tight')
                plt.close()
                plot_files.append(plot_file)
        
        # If we have comparison data, create comparison plots
        if self.comparison_results:
            # Prepare combined dataframe for comparison
            comparison_dfs = [self.results.assign(cache_lib='HCache')]
            for lib_name, lib_df in self.comparison_results.items():
                comparison_dfs.append(lib_df)
            
            combined_df = pd.concat(comparison_dfs, ignore_index=True)
            
            # Plot comparison for each benchmark type
            for benchmark_type in benchmark_types:
                benchmark_data = combined_df[combined_df['name'] == benchmark_type]
                
                if len(benchmark_data) > 0 and 'ValueSize' in benchmark_data.columns:
                    # Performance comparison
                    plt.figure(figsize=(12, 7))
                    sns.barplot(x='ValueSize', y='ns_per_op', hue='cache_lib', data=benchmark_data)
                    plt.title(f'{benchmark_type} - Performance Comparison')
                    plt.xlabel('Value Size (bytes)')
                    plt.ylabel('Time per Operation (ns)')
                    plt.yscale('log')
                    plt.grid(True, alpha=0.3)
                    plt.legend(title='Cache Library')
                    
                    # Save plot
                    plot_file = os.path.join(self.img_dir, f'{benchmark_type}_comparison.png')
                    plt.savefig(plot_file, dpi=300, bbox_inches='tight')
                    plt.close()
                    plot_files.append(plot_file)
                    
                    # Interactive comparison plot
                    fig = px.bar(
                        benchmark_data, 
                        x='ValueSize', 
                        y='ns_per_op',
                        color='cache_lib',
                        barmode='group',
                        title=f'{benchmark_type} - Performance Comparison',
                        labels={'ValueSize': 'Value Size (bytes)', 'ns_per_op': 'Time per Operation (ns)', 'cache_lib': 'Cache Library'},
                        log_y=True
                    )
                    
                    html_file = os.path.join(self.html_dir, f'{benchmark_type}_comparison.html')
                    fig.write_html(html_file)
                    plot_files.append(html_file)
        
        return plot_files
    
    def generate_summary_report(self):
        """
        Generate a comprehensive summary report in markdown format.
        
        Returns:
            Path to the generated report file
        """
        if self.results is None:
            raise ValueError("No benchmark results loaded")
        
        # Generate timestamp
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        
        # Start building the report
        report = [
            "# HCache Benchmark Analysis Report",
            f"Generated on: {timestamp}\n",
            "## Summary Statistics",
        ]
        
        # Add summary statistics
        summary = self.results.groupby('name').agg({
            'ns_per_op': ['mean', 'median', 'min', 'max'],
            'bytes_per_op': ['mean'],
            'allocs_per_op': ['mean']
        })
        
        report.append("```")
        report.append(str(summary))
        report.append("```\n")
        
        # Add performance comparison section
        report.append("## Performance Comparison")
        report.append("### Latency Comparison (ns/op)")
        
        # Create a summary table for latency
        latency_table = self.results.pivot_table(
            index=['name'], 
            columns=['ValueSize'], 
            values='ns_per_op',
            aggfunc='mean'
        )
        
        report.append("```")
        report.append(str(latency_table))
        report.append("```\n")
        
        # Add memory allocation section
        report.append("### Memory Allocation Comparison (allocs/op)")
        
        # Create a summary table for allocations
        allocs_table = self.results.pivot_table(
            index=['name'], 
            columns=['ValueSize'], 
            values='allocs_per_op',
            aggfunc='mean'
        )
        
        report.append("```")
        report.append(str(allocs_table))
        report.append("```\n")
        
        # Add comparison with other libraries if available
        if self.comparison_results:
            report.append("## Comparison with Other Cache Libraries")
            
            # Create a combined dataframe for comparison
            comparison_dfs = [self.results.assign(cache_lib='HCache')]
            for lib_name, lib_df in self.comparison_results.items():
                comparison_dfs.append(lib_df)
            
            combined_df = pd.concat(comparison_dfs, ignore_index=True)
            
            # Create comparison tables
            for metric in ['ns_per_op', 'bytes_per_op', 'allocs_per_op']:
                metric_name = {
                    'ns_per_op': 'Latency (ns/op)',
                    'bytes_per_op': 'Memory Usage (B/op)',
                    'allocs_per_op': 'Allocations (allocs/op)'
                }.get(metric, metric)
                
                report.append(f"### {metric_name} Comparison")
                
                comparison_table = combined_df.pivot_table(
                    index=['name', 'ValueSize'],
                    columns=['cache_lib'],
                    values=metric,
                    aggfunc='mean'
                )
                
                report.append("```")
                report.append(str(comparison_table))
                report.append("```\n")
                
                # Calculate percentage improvement over other libraries
                if 'HCache' in comparison_table.columns:
                    report.append("#### Percentage Improvement")
                    
                    improvement_dfs = []
                    for lib in comparison_table.columns:
                        if lib != 'HCache':
                            # Calculate improvement percentage
                            improvement = ((comparison_table[lib] - comparison_table['HCache']) / comparison_table[lib]) * 100
                            improvement.name = f"vs_{lib}(%)"
                            improvement_dfs.append(improvement)
                    
                    if improvement_dfs:
                        improvement_table = pd.concat(improvement_dfs, axis=1)
                        
                        report.append("```")
                        report.append(str(improvement_table))
                        report.append("```\n")
                        
                        report.append("*Positive values indicate HCache is faster/better*\n")
        
        # Add conclusion
        report.append("## Conclusion")
        report.append("Based on the benchmark results, we can draw the following conclusions:")
        report.append("")
        report.append("1. **Performance**: [Add conclusions about performance]")
        report.append("2. **Memory Usage**: [Add conclusions about memory usage]")
        report.append("3. **Allocations**: [Add conclusions about allocations]")
        report.append("4. **Comparison**: [Add conclusions about comparison with other libraries]")
        
        # Write report to file
        report_content = "\n".join(report)
        report_file = os.path.join(self.report_dir, f'benchmark_report_{datetime.now().strftime("%Y%m%d")}.md')
        
        with open(report_file, 'w') as f:
            f.write(report_content)
        
        return report_file

def main():
    """Main function to run the benchmark analyzer."""
    parser = argparse.ArgumentParser(description='Analyze Go benchmark results')
    parser.add_argument('--input', '-i', required=True, help='Directory containing benchmark result files')
    parser.add_argument('--output', '-o', required=True, help='Directory to save analysis results')
    parser.add_argument('--comparison', '-c', help='Directory containing benchmark results for other cache libraries')
    parser.add_argument('--pattern', '-p', help='Regex pattern to filter input files')
    
    args = parser.parse_args()
    
    analyzer = BenchmarkAnalyzer(args.input, args.output)
    
    print("Loading benchmark results...")
    results = analyzer.load_benchmark_results(args.pattern)
    print(f"Loaded {len(results)} benchmark results")
    
    if args.comparison:
        print("Loading comparison data...")
        comparison_results = analyzer.load_comparison_data(args.comparison)
        print(f"Loaded comparison data for {len(comparison_results)} cache libraries")
    
    print("Generating descriptive statistics...")
    stats, percentiles = analyzer.generate_descriptive_stats()
    
    print("Generating performance comparison plots...")
    plot_files = analyzer.plot_performance_comparison()
    print(f"Generated {len(plot_files)} plot files")
    
    print("Generating summary report...")
    report_file = analyzer.generate_summary_report()
    print(f"Generated summary report: {report_file}")
    
    print("Analysis complete!")

if __name__ == "__main__":
    main() 