#!/usr/bin/env python3
"""
HCache Concurrency Analysis Tool

This script processes concurrency test results from vegeta load tests and generates
visualizations and analysis.
"""

import os
import re
import json
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
import plotly.express as px
import plotly.graph_objects as go
from plotly.subplots import make_subplots
from datetime import datetime
import argparse

# Set style for matplotlib
plt.style.use('ggplot')
sns.set_theme(style="whitegrid")

class ConcurrencyAnalyzer:
    """Analyzes concurrency test results and generates visualizations."""
    
    def __init__(self, input_dir, output_dir):
        """
        Initialize the analyzer with input and output directories.
        
        Args:
            input_dir: Directory containing concurrency test result files
            output_dir: Directory to save analysis results
        """
        self.input_dir = input_dir
        self.output_dir = output_dir
        self.results = None
        
        # Create output directory if it doesn't exist
        os.makedirs(output_dir, exist_ok=True)
        
        # Create subdirectories for different output types
        self.img_dir = os.path.join(output_dir, 'images')
        self.csv_dir = os.path.join(output_dir, 'csv')
        self.html_dir = os.path.join(output_dir, 'html')
        self.report_dir = os.path.join(output_dir, 'reports')
        
        for directory in [self.img_dir, self.csv_dir, self.html_dir, self.report_dir]:
            os.makedirs(directory, exist_ok=True)
    
    def parse_vegeta_results(self, filename):
        """
        Parse a Vegeta JSON result file.
        
        Args:
            filename: Path to the Vegeta result file
            
        Returns:
            DataFrame containing parsed concurrency test results
        """
        with open(filename, 'r') as f:
            content = f.read()
        
        # Parse JSON content
        try:
            data = json.loads(content)
        except json.JSONDecodeError:
            # Try to parse line-by-line (Vegeta can output one JSON object per line)
            data = []
            for line in content.strip().split('\n'):
                if line.strip():
                    try:
                        data.append(json.loads(line))
                    except json.JSONDecodeError:
                        pass
        
        # Check if data is a list or a single object
        if not isinstance(data, list):
            data = [data]
        
        # Extract relevant metrics
        results = []
        
        for item in data:
            # Extract test parameters from filename or data
            test_params = self.extract_test_params(filename, item)
            
            # Extract metrics
            metrics = {
                'latency_mean': item.get('latencies', {}).get('mean', 0) / 1e6,  # ns to ms
                'latency_p50': item.get('latencies', {}).get('50th', 0) / 1e6,   # ns to ms
                'latency_p90': item.get('latencies', {}).get('90th', 0) / 1e6,   # ns to ms
                'latency_p95': item.get('latencies', {}).get('95th', 0) / 1e6,   # ns to ms
                'latency_p99': item.get('latencies', {}).get('99th', 0) / 1e6,   # ns to ms
                'latency_max': item.get('latencies', {}).get('max', 0) / 1e6,    # ns to ms
                'throughput': item.get('throughput', 0),
                'success_rate': 100 * (1 - item.get('success', 0)),
                'requests': item.get('requests', 0),
                'duration': item.get('duration', 0) / 1e9,  # ns to s
                'errors': item.get('errors', 0),
                'rate': item.get('rate', 0)
            }
            
            # Combine parameters and metrics
            result = {**test_params, **metrics}
            results.append(result)
        
        return pd.DataFrame(results)
    
    def extract_test_params(self, filename, data):
        """
        Extract test parameters from filename or data.
        
        Args:
            filename: Path to the result file
            data: Parsed JSON data
            
        Returns:
            Dictionary containing test parameters
        """
        params = {}
        
        # Extract from filename
        basename = os.path.basename(filename)
        
        # Try to extract concurrency level
        concurrency_match = re.search(r'c(\d+)', basename)
        if concurrency_match:
            params['concurrency'] = int(concurrency_match.group(1))
        else:
            params['concurrency'] = 1  # Default
        
        # Try to extract rate
        rate_match = re.search(r'r(\d+)', basename)
        if rate_match:
            params['target_rate'] = int(rate_match.group(1))
        else:
            params['target_rate'] = data.get('rate', 0)
        
        # Try to extract duration
        duration_match = re.search(r'd(\d+)([smh])', basename)
        if duration_match:
            value = int(duration_match.group(1))
            unit = duration_match.group(2)
            
            if unit == 's':
                params['target_duration'] = value
            elif unit == 'm':
                params['target_duration'] = value * 60
            elif unit == 'h':
                params['target_duration'] = value * 3600
        else:
            params['target_duration'] = data.get('duration', 0) / 1e9  # ns to s
        
        # Try to extract cache configuration
        cache_match = re.search(r'cache-(\w+)', basename)
        if cache_match:
            params['cache_config'] = cache_match.group(1)
        else:
            params['cache_config'] = 'default'
        
        # Try to extract test date
        date_match = re.search(r'(\d{8})', basename)
        if date_match:
            params['test_date'] = date_match.group(1)
        else:
            params['test_date'] = '00000000'
        
        return params
    
    def load_concurrency_results(self, pattern=None):
        """
        Load all concurrency test results from the input directory.
        
        Args:
            pattern: Optional regex pattern to filter files
            
        Returns:
            DataFrame containing all concurrency test results
        """
        all_results = []
        
        for filename in os.listdir(self.input_dir):
            if not (filename.endswith('.json') or filename.endswith('.vegeta')):
                continue
            
            if pattern and not re.search(pattern, filename):
                continue
            
            filepath = os.path.join(self.input_dir, filename)
            
            try:
                df = self.parse_vegeta_results(filepath)
                df['source_file'] = filename
                
                # Convert test_date to datetime
                if 'test_date' in df.columns:
                    df['date'] = pd.to_datetime(df['test_date'], format='%Y%m%d')
                
                all_results.append(df)
            except Exception as e:
                print(f"Error parsing file {filename}: {e}")
        
        if not all_results:
            raise ValueError(f"No concurrency test results found in {self.input_dir}")
        
        self.results = pd.concat(all_results, ignore_index=True)
        return self.results
    
    def generate_descriptive_stats(self):
        """
        Generate descriptive statistics for concurrency test results.
        
        Returns:
            DataFrame containing descriptive statistics
        """
        if self.results is None:
            raise ValueError("No concurrency test results loaded")
        
        # Group by test parameters and calculate stats
        stats = self.results.groupby(['concurrency', 'target_rate', 'cache_config']).agg({
            'latency_mean': ['mean', 'median', 'std', 'min', 'max'],
            'latency_p95': ['mean', 'median'],
            'latency_p99': ['mean', 'median'],
            'throughput': ['mean', 'median', 'std', 'min', 'max'],
            'success_rate': ['mean', 'min']
        }).reset_index()
        
        # Save stats to CSV
        stats_file = os.path.join(self.csv_dir, 'concurrency_stats.csv')
        stats.to_csv(stats_file, index=False)
        
        return stats
    
    def plot_concurrency_results(self):
        """
        Generate concurrency test result plots.
        
        Returns:
            List of paths to generated plot files
        """
        if self.results is None:
            raise ValueError("No concurrency test results loaded")
        
        plot_files = []
        
        # Plot latency by concurrency level
        if len(self.results['concurrency'].unique()) > 1:
            plt.figure(figsize=(12, 8))
            sns.lineplot(x='concurrency', y='latency_mean', hue='cache_config', data=self.results, marker='o')
            plt.title('Mean Latency by Concurrency Level')
            plt.xlabel('Concurrency Level')
            plt.ylabel('Mean Latency (ms)')
            plt.grid(True, alpha=0.3)
            
            # Save plot
            plot_file = os.path.join(self.img_dir, 'latency_by_concurrency.png')
            plt.savefig(plot_file, dpi=300, bbox_inches='tight')
            plt.close()
            plot_files.append(plot_file)
            
            # Interactive plot with Plotly
            fig = px.line(
                self.results, 
                x='concurrency', 
                y='latency_mean',
                color='cache_config',
                markers=True,
                title='Mean Latency by Concurrency Level',
                labels={'concurrency': 'Concurrency Level', 'latency_mean': 'Mean Latency (ms)', 'cache_config': 'Cache Configuration'}
            )
            
            html_file = os.path.join(self.html_dir, 'latency_by_concurrency.html')
            fig.write_html(html_file)
            plot_files.append(html_file)
            
            # Plot percentile latencies
            plt.figure(figsize=(14, 8))
            
            # Melt the dataframe to get latency percentiles in one column
            latency_cols = ['latency_p50', 'latency_p90', 'latency_p95', 'latency_p99']
            latency_df = pd.melt(
                self.results, 
                id_vars=['concurrency', 'cache_config'], 
                value_vars=latency_cols,
                var_name='percentile', 
                value_name='latency'
            )
            
            # Plot
            sns.lineplot(x='concurrency', y='latency', hue='percentile', style='cache_config', data=latency_df, marker='o')
            plt.title('Latency Percentiles by Concurrency Level')
            plt.xlabel('Concurrency Level')
            plt.ylabel('Latency (ms)')
            plt.grid(True, alpha=0.3)
            
            # Save plot
            plot_file = os.path.join(self.img_dir, 'latency_percentiles.png')
            plt.savefig(plot_file, dpi=300, bbox_inches='tight')
            plt.close()
            plot_files.append(plot_file)
        
        # Plot throughput by concurrency level
        if len(self.results['concurrency'].unique()) > 1:
            plt.figure(figsize=(12, 8))
            sns.lineplot(x='concurrency', y='throughput', hue='cache_config', data=self.results, marker='o')
            plt.title('Throughput by Concurrency Level')
            plt.xlabel('Concurrency Level')
            plt.ylabel('Throughput (req/s)')
            plt.grid(True, alpha=0.3)
            
            # Save plot
            plot_file = os.path.join(self.img_dir, 'throughput_by_concurrency.png')
            plt.savefig(plot_file, dpi=300, bbox_inches='tight')
            plt.close()
            plot_files.append(plot_file)
            
            # Interactive plot with Plotly
            fig = px.line(
                self.results, 
                x='concurrency', 
                y='throughput',
                color='cache_config',
                markers=True,
                title='Throughput by Concurrency Level',
                labels={'concurrency': 'Concurrency Level', 'throughput': 'Throughput (req/s)', 'cache_config': 'Cache Configuration'}
            )
            
            html_file = os.path.join(self.html_dir, 'throughput_by_concurrency.html')
            fig.write_html(html_file)
            plot_files.append(html_file)
        
        # Plot success rate by concurrency level
        if len(self.results['concurrency'].unique()) > 1:
            plt.figure(figsize=(12, 8))
            sns.lineplot(x='concurrency', y='success_rate', hue='cache_config', data=self.results, marker='o')
            plt.title('Success Rate by Concurrency Level')
            plt.xlabel('Concurrency Level')
            plt.ylabel('Success Rate (%)')
            plt.grid(True, alpha=0.3)
            
            # Save plot
            plot_file = os.path.join(self.img_dir, 'success_by_concurrency.png')
            plt.savefig(plot_file, dpi=300, bbox_inches='tight')
            plt.close()
            plot_files.append(plot_file)
        
        # Create a latency vs throughput scatter plot
        plt.figure(figsize=(12, 8))
        scatter = sns.scatterplot(
            x='throughput', 
            y='latency_mean', 
            hue='cache_config', 
            size='concurrency', 
            data=self.results,
            sizes=(50, 200)
        )
        plt.title('Latency vs Throughput')
        plt.xlabel('Throughput (req/s)')
        plt.ylabel('Mean Latency (ms)')
        plt.grid(True, alpha=0.3)
        
        # Add annotations for concurrency levels
        for _, row in self.results.iterrows():
            plt.annotate(
                f"c={row['concurrency']}", 
                (row['throughput'], row['latency_mean']),
                textcoords="offset points",
                xytext=(0, 5),
                ha='center'
            )
        
        # Save plot
        plot_file = os.path.join(self.img_dir, 'latency_vs_throughput.png')
        plt.savefig(plot_file, dpi=300, bbox_inches='tight')
        plt.close()
        plot_files.append(plot_file)
        
        # Interactive scatter plot with Plotly
        fig = px.scatter(
            self.results, 
            x='throughput', 
            y='latency_mean',
            color='cache_config',
            size='concurrency',
            hover_data=['concurrency', 'target_rate', 'success_rate'],
            title='Latency vs Throughput',
            labels={
                'throughput': 'Throughput (req/s)', 
                'latency_mean': 'Mean Latency (ms)', 
                'cache_config': 'Cache Configuration',
                'concurrency': 'Concurrency Level'
            }
        )
        
        html_file = os.path.join(self.html_dir, 'latency_vs_throughput.html')
        fig.write_html(html_file)
        plot_files.append(html_file)
        
        # Create a box plot of latencies by cache configuration
        plt.figure(figsize=(14, 8))
        
        # Melt the dataframe to get latency metrics in one column
        latency_cols = ['latency_mean', 'latency_p50', 'latency_p90', 'latency_p95', 'latency_p99']
        latency_df = pd.melt(
            self.results, 
            id_vars=['cache_config', 'concurrency'], 
            value_vars=latency_cols,
            var_name='metric', 
            value_name='latency'
        )
        
        # Plot
        sns.boxplot(x='cache_config', y='latency', hue='metric', data=latency_df)
        plt.title('Latency Distribution by Cache Configuration')
        plt.xlabel('Cache Configuration')
        plt.ylabel('Latency (ms)')
        plt.grid(True, alpha=0.3)
        plt.legend(title='Latency Metric')
        
        # Save plot
        plot_file = os.path.join(self.img_dir, 'latency_boxplot.png')
        plt.savefig(plot_file, dpi=300, bbox_inches='tight')
        plt.close()
        plot_files.append(plot_file)
        
        return plot_files
    
    def generate_summary_report(self):
        """
        Generate a comprehensive summary report in markdown format.
        
        Returns:
            Path to the generated report file
        """
        if self.results is None:
            raise ValueError("No concurrency test results loaded")
        
        # Generate timestamp
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        
        # Start building the report
        report = [
            "# HCache Concurrency Analysis Report",
            f"Generated on: {timestamp}\n",
            "## Summary Statistics",
        ]
        
        # Add summary statistics
        summary = self.results.groupby(['cache_config']).agg({
            'latency_mean': ['mean', 'median', 'min', 'max'],
            'latency_p95': ['mean'],
            'latency_p99': ['mean'],
            'throughput': ['mean', 'max'],
            'success_rate': ['mean', 'min']
        })
        
        report.append("```")
        report.append(str(summary))
        report.append("```\n")
        
        # Add concurrency level comparison
        if len(self.results['concurrency'].unique()) > 1:
            report.append("## Performance by Concurrency Level")
            
            concurrency_table = self.results.pivot_table(
                index=['cache_config'], 
                columns=['concurrency'], 
                values=['latency_mean', 'throughput', 'success_rate'],
                aggfunc='mean'
            )
            
            report.append("### Mean Latency (ms) by Concurrency Level")
            report.append("```")
            report.append(str(concurrency_table['latency_mean']))
            report.append("```\n")
            
            report.append("### Throughput (req/s) by Concurrency Level")
            report.append("```")
            report.append(str(concurrency_table['throughput']))
            report.append("```\n")
            
            report.append("### Success Rate (%) by Concurrency Level")
            report.append("```")
            report.append(str(concurrency_table['success_rate']))
            report.append("```\n")
        
        # Add latency percentile comparison
        report.append("## Latency Percentiles by Cache Configuration")
        
        percentile_table = self.results.groupby(['cache_config']).agg({
            'latency_p50': 'mean',
            'latency_p90': 'mean',
            'latency_p95': 'mean',
            'latency_p99': 'mean',
            'latency_max': 'mean'
        })
        
        report.append("```")
        report.append(str(percentile_table))
        report.append("```\n")
        
        # Find optimal concurrency level for each cache configuration
        report.append("## Optimal Concurrency Level")
        report.append("")
        
        for cache_config in self.results['cache_config'].unique():
            config_data = self.results[self.results['cache_config'] == cache_config]
            
            # Find the concurrency level with the highest throughput
            max_throughput_row = config_data.loc[config_data['throughput'].idxmax()]
            
            # Find the concurrency level with the lowest latency
            min_latency_row = config_data.loc[config_data['latency_mean'].idxmin()]
            
            report.append(f"### For {cache_config} configuration:")
            report.append(f"- Highest throughput: **{max_throughput_row['throughput']:.2f} req/s** at concurrency level {max_throughput_row['concurrency']}")
            report.append(f"- Lowest latency: **{min_latency_row['latency_mean']:.2f} ms** at concurrency level {min_latency_row['concurrency']}")
            report.append(f"- Recommended concurrency level: **{max_throughput_row['concurrency']}** (optimizing for throughput)")
            report.append("")
        
        # Add conclusion
        report.append("## Conclusion")
        report.append("Based on the concurrency test results, we can draw the following conclusions:")
        report.append("")
        
        # Calculate average improvement across concurrency levels
        if len(self.results['cache_config'].unique()) > 1:
            baseline_config = self.results['cache_config'].iloc[0]
            
            for cache_config in self.results['cache_config'].unique():
                if cache_config == baseline_config:
                    continue
                
                # Calculate average improvement in latency and throughput
                comparison = self.results.pivot_table(
                    index=['concurrency'],
                    columns=['cache_config'],
                    values=['latency_mean', 'throughput'],
                    aggfunc='mean'
                )
                
                latency_improvement = ((comparison['latency_mean'][baseline_config] - comparison['latency_mean'][cache_config]) / 
                                      comparison['latency_mean'][baseline_config] * 100).mean()
                
                throughput_improvement = ((comparison['throughput'][cache_config] - comparison['throughput'][baseline_config]) / 
                                        comparison['throughput'][baseline_config] * 100).mean()
                
                report.append(f"1. **{cache_config}** vs **{baseline_config}**:")
                report.append(f"   - Average latency improvement: **{latency_improvement:.2f}%**")
                report.append(f"   - Average throughput improvement: **{throughput_improvement:.2f}%**")
        
        # Add general conclusions
        report.append("")
        report.append("### General Observations:")
        report.append("- Performance scales with concurrency up to a certain point, after which latency increases and throughput plateaus.")
        report.append("- The optimal concurrency level depends on the specific cache configuration and hardware.")
        report.append("- Higher concurrency levels may lead to increased resource contention and reduced performance.")
        
        # Write report to file
        report_content = "\n".join(report)
        report_file = os.path.join(self.report_dir, f'concurrency_report_{datetime.now().strftime("%Y%m%d")}.md')
        
        with open(report_file, 'w') as f:
            f.write(report_content)
        
        return report_file

def main():
    """Main function to run the concurrency analyzer."""
    parser = argparse.ArgumentParser(description='Analyze concurrency test results')
    parser.add_argument('--input', '-i', required=True, help='Directory containing concurrency test result files')
    parser.add_argument('--output', '-o', required=True, help='Directory to save analysis results')
    parser.add_argument('--pattern', '-p', help='Regex pattern to filter input files')
    
    args = parser.parse_args()
    
    analyzer = ConcurrencyAnalyzer(args.input, args.output)
    
    print("Loading concurrency test results...")
    results = analyzer.load_concurrency_results(args.pattern)
    print(f"Loaded {len(results)} concurrency test results")
    
    print("Generating descriptive statistics...")
    stats = analyzer.generate_descriptive_stats()
    
    print("Generating concurrency test plots...")
    plot_files = analyzer.plot_concurrency_results()
    print(f"Generated {len(plot_files)} plot files")
    
    print("Generating summary report...")
    report_file = analyzer.generate_summary_report()
    print(f"Generated summary report: {report_file}")
    
    print("Analysis complete!")

if __name__ == "__main__":
    main()