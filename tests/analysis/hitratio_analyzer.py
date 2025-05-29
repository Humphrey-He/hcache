#!/usr/bin/env python3
"""
HCache Hit Ratio Analysis Tool

This script processes hit ratio test results and generates visualizations and analysis.
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

class HitRatioAnalyzer:
    """Analyzes cache hit ratio test results and generates visualizations."""
    
    def __init__(self, input_dir, output_dir):
        """
        Initialize the analyzer with input and output directories.
        
        Args:
            input_dir: Directory containing hit ratio test result files
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
    
    def parse_test_output(self, filename):
        """
        Parse a Go test output file containing hit ratio test results.
        
        Args:
            filename: Path to the test output file
            
        Returns:
            DataFrame containing parsed hit ratio results
        """
        with open(filename, 'r') as f:
            content = f.read()
        
        # Regular expressions to extract test results
        test_pattern = r'=== RUN\s+Test(\w+)'
        result_pattern = r'--- PASS: Test(\w+)\s+\(([0-9.]+)s\)'
        
        # Pattern to extract hit ratio test results
        hitratio_pattern = r'测试结果:.*?总操作数:\s+(\d+).*?命中数:\s+(\d+).*?未命中数:\s+(\d+).*?命中率:\s+([\d.]+)%.*?淘汰数:\s+(\d+).*?淘汰比率:\s+([\d.]+)%.*?持续时间:\s+([0-9.]+[µnm]?s)'
        
        # Extract test names and results
        tests = re.findall(test_pattern, content)
        results = re.findall(result_pattern, content)
        
        # Extract hit ratio specific results
        hitratio_results = re.findall(hitratio_pattern, content, re.DOTALL)
        
        # Parse results into structured data
        parsed_results = []
        
        for match in hitratio_results:
            total_ops = int(match[0])
            hits = int(match[1])
            misses = int(match[2])
            hit_ratio = float(match[3])
            evictions = int(match[4])
            eviction_ratio = float(match[5])
            duration_str = match[6]
            
            # Parse test name and parameters
            test_name = None
            cache_size = None
            distribution = None
            policy = None
            
            # Look for the test name in the surrounding context
            context_before = content.split(f"总操作数: {total_ops}")[0]
            test_context = context_before.split("=== RUN")[-1]
            
            # Extract test name
            name_match = re.search(r'Test(\w+)', test_context)
            if name_match:
                test_name = name_match.group(1)
            
            # Try to extract parameters from test name or surrounding context
            if "ZipfLow" in test_context:
                distribution = "zipf-1.07"
            elif "ZipfHigh" in test_context:
                distribution = "zipf-1.2"
            elif "Uniform" in test_context:
                distribution = "uniform"
            
            if "LRU" in test_context:
                policy = "lru"
            elif "LFU" in test_context:
                policy = "lfu"
            elif "FIFO" in test_context:
                policy = "fifo"
            elif "Random" in test_context:
                policy = "random"
            
            # Extract cache size
            size_match = re.search(r'Size(\d+)', test_context)
            if size_match:
                cache_size = int(size_match.group(1))
            else:
                # Default sizes from the test file
                if "small" in test_context.lower():
                    cache_size = 1000
                elif "large" in test_context.lower():
                    cache_size = 100000
                else:
                    cache_size = 10000  # medium size default
            
            # Parse duration to milliseconds
            duration_ms = 0
            if 'µs' in duration_str:
                duration_ms = float(duration_str.replace('µs', '')) / 1000
            elif 'ns' in duration_str:
                duration_ms = float(duration_str.replace('ns', '')) / 1000000
            elif 'ms' in duration_str:
                duration_ms = float(duration_str.replace('ms', ''))
            elif 's' in duration_str:
                duration_ms = float(duration_str.replace('s', '')) * 1000
            
            result = {
                'test_name': test_name,
                'cache_size': cache_size,
                'distribution': distribution,
                'policy': policy,
                'total_operations': total_ops,
                'hits': hits,
                'misses': misses,
                'hit_ratio': hit_ratio,
                'evictions': evictions,
                'eviction_ratio': eviction_ratio,
                'duration_ms': duration_ms
            }
            
            parsed_results.append(result)
        
        return pd.DataFrame(parsed_results)
    
    def load_test_results(self, pattern=None):
        """
        Load all hit ratio test results from the input directory.
        
        Args:
            pattern: Optional regex pattern to filter files
            
        Returns:
            DataFrame containing all hit ratio test results
        """
        all_results = []
        
        for filename in os.listdir(self.input_dir):
            if not (filename.endswith('.txt') or filename.endswith('.log')):
                continue
            
            if pattern and not re.search(pattern, filename):
                continue
            
            filepath = os.path.join(self.input_dir, filename)
            
            # Extract metadata from filename
            match = re.search(r'hitratio_(\d{8})\.(?:txt|log)', filename)
            if match:
                date_str = match.group(1)
            else:
                date_str = '00000000'
            
            try:
                df = self.parse_test_output(filepath)
                
                # Add metadata columns
                df['date'] = pd.to_datetime(date_str, format='%Y%m%d')
                df['source_file'] = filename
                
                all_results.append(df)
            except Exception as e:
                print(f"Error parsing file {filename}: {e}")
        
        if not all_results:
            raise ValueError(f"No hit ratio test results found in {self.input_dir}")
        
        self.results = pd.concat(all_results, ignore_index=True)
        return self.results
    
    def generate_descriptive_stats(self):
        """
        Generate descriptive statistics for hit ratio results.
        
        Returns:
            DataFrame containing descriptive statistics
        """
        if self.results is None:
            raise ValueError("No hit ratio test results loaded")
        
        # Group by test parameters and calculate stats
        stats = self.results.groupby(['distribution', 'policy', 'cache_size']).agg({
            'hit_ratio': ['mean', 'median', 'std', 'min', 'max'],
            'eviction_ratio': ['mean', 'median', 'std'],
            'duration_ms': ['mean', 'median']
        }).reset_index()
        
        # Save stats to CSV
        stats_file = os.path.join(self.csv_dir, 'hitratio_stats.csv')
        stats.to_csv(stats_file, index=False)
        
        return stats
    
    def plot_hit_ratio_comparison(self):
        """
        Generate hit ratio comparison plots.
        
        Returns:
            List of paths to generated plot files
        """
        if self.results is None:
            raise ValueError("No hit ratio test results loaded")
        
        plot_files = []
        
        # Plot hit ratio by distribution type
        plt.figure(figsize=(12, 8))
        sns.barplot(x='distribution', y='hit_ratio', hue='policy', data=self.results)
        plt.title('Hit Ratio by Distribution Type and Eviction Policy')
        plt.xlabel('Distribution Type')
        plt.ylabel('Hit Ratio (%)')
        plt.grid(True, alpha=0.3)
        plt.legend(title='Eviction Policy')
        
        # Save plot
        plot_file = os.path.join(self.img_dir, 'hitratio_by_distribution.png')
        plt.savefig(plot_file, dpi=300, bbox_inches='tight')
        plt.close()
        plot_files.append(plot_file)
        
        # Interactive plot with Plotly
        fig = px.bar(
            self.results, 
            x='distribution', 
            y='hit_ratio',
            color='policy',
            barmode='group',
            title='Hit Ratio by Distribution Type and Eviction Policy',
            labels={'distribution': 'Distribution Type', 'hit_ratio': 'Hit Ratio (%)', 'policy': 'Eviction Policy'}
        )
        
        html_file = os.path.join(self.html_dir, 'hitratio_by_distribution.html')
        fig.write_html(html_file)
        plot_files.append(html_file)
        
        # Plot hit ratio by cache size
        plt.figure(figsize=(12, 8))
        sns.lineplot(x='cache_size', y='hit_ratio', hue='policy', style='distribution', data=self.results, markers=True)
        plt.title('Hit Ratio by Cache Size')
        plt.xlabel('Cache Size (entries)')
        plt.ylabel('Hit Ratio (%)')
        plt.xscale('log')
        plt.grid(True, alpha=0.3)
        plt.legend(title='Policy / Distribution')
        
        # Save plot
        plot_file = os.path.join(self.img_dir, 'hitratio_by_size.png')
        plt.savefig(plot_file, dpi=300, bbox_inches='tight')
        plt.close()
        plot_files.append(plot_file)
        
        # Interactive plot with Plotly
        fig = px.line(
            self.results, 
            x='cache_size', 
            y='hit_ratio',
            color='policy',
            line_dash='distribution',
            markers=True,
            title='Hit Ratio by Cache Size',
            labels={'cache_size': 'Cache Size (entries)', 'hit_ratio': 'Hit Ratio (%)', 'policy': 'Eviction Policy'},
            log_x=True
        )
        
        html_file = os.path.join(self.html_dir, 'hitratio_by_size.html')
        fig.write_html(html_file)
        plot_files.append(html_file)
        
        # Plot eviction ratio by policy
        plt.figure(figsize=(12, 8))
        sns.barplot(x='policy', y='eviction_ratio', hue='distribution', data=self.results)
        plt.title('Eviction Ratio by Policy and Distribution')
        plt.xlabel('Eviction Policy')
        plt.ylabel('Eviction Ratio (%)')
        plt.grid(True, alpha=0.3)
        plt.legend(title='Distribution')
        
        # Save plot
        plot_file = os.path.join(self.img_dir, 'eviction_by_policy.png')
        plt.savefig(plot_file, dpi=300, bbox_inches='tight')
        plt.close()
        plot_files.append(plot_file)
        
        # Create a heatmap of hit ratio by policy and distribution
        pivot_data = self.results.pivot_table(
            index='policy', 
            columns='distribution', 
            values='hit_ratio',
            aggfunc='mean'
        )
        
        plt.figure(figsize=(10, 8))
        sns.heatmap(pivot_data, annot=True, fmt='.1f', cmap='YlGnBu')
        plt.title('Hit Ratio Heatmap by Policy and Distribution')
        plt.ylabel('Eviction Policy')
        plt.xlabel('Distribution')
        
        # Save plot
        plot_file = os.path.join(self.img_dir, 'hitratio_heatmap.png')
        plt.savefig(plot_file, dpi=300, bbox_inches='tight')
        plt.close()
        plot_files.append(plot_file)
        
        # Create a 3D surface plot of hit ratio by policy, distribution and cache size
        if len(self.results['cache_size'].unique()) > 1:
            pivot_3d = self.results.pivot_table(
                index='policy', 
                columns=['distribution', 'cache_size'], 
                values='hit_ratio',
                aggfunc='mean'
            )
            
            # Create a 3D surface plot with Plotly
            fig = go.Figure()
            
            for policy in self.results['policy'].unique():
                for dist in self.results['distribution'].unique():
                    policy_dist_data = self.results[(self.results['policy'] == policy) & 
                                                  (self.results['distribution'] == dist)]
                    
                    if len(policy_dist_data) > 0:
                        fig.add_trace(go.Scatter3d(
                            x=policy_dist_data['cache_size'],
                            y=[dist] * len(policy_dist_data),
                            z=policy_dist_data['hit_ratio'],
                            mode='markers+lines',
                            name=f'{policy} - {dist}',
                            marker=dict(size=8)
                        ))
            
            fig.update_layout(
                title='Hit Ratio by Policy, Distribution and Cache Size',
                scene=dict(
                    xaxis_title='Cache Size',
                    yaxis_title='Distribution',
                    zaxis_title='Hit Ratio (%)',
                    xaxis=dict(type='log')
                ),
                width=1000,
                height=800
            )
            
            html_file = os.path.join(self.html_dir, 'hitratio_3d.html')
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
            raise ValueError("No hit ratio test results loaded")
        
        # Generate timestamp
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        
        # Start building the report
        report = [
            "# HCache Hit Ratio Analysis Report",
            f"Generated on: {timestamp}\n",
            "## Summary Statistics",
        ]
        
        # Add summary statistics
        summary = self.results.groupby(['distribution', 'policy']).agg({
            'hit_ratio': ['mean', 'median', 'min', 'max'],
            'eviction_ratio': ['mean'],
            'duration_ms': ['mean']
        })
        
        report.append("```")
        report.append(str(summary))
        report.append("```\n")
        
        # Add hit ratio comparison by distribution
        report.append("## Hit Ratio by Distribution Type")
        
        dist_table = self.results.pivot_table(
            index=['policy'], 
            columns=['distribution'], 
            values='hit_ratio',
            aggfunc='mean'
        )
        
        report.append("```")
        report.append(str(dist_table))
        report.append("```\n")
        
        # Add hit ratio comparison by cache size
        if len(self.results['cache_size'].unique()) > 1:
            report.append("## Hit Ratio by Cache Size")
            
            size_table = self.results.pivot_table(
                index=['policy', 'distribution'], 
                columns=['cache_size'], 
                values='hit_ratio',
                aggfunc='mean'
            )
            
            report.append("```")
            report.append(str(size_table))
            report.append("```\n")
        
        # Add eviction ratio comparison
        report.append("## Eviction Ratio by Policy")
        
        eviction_table = self.results.pivot_table(
            index=['distribution'], 
            columns=['policy'], 
            values='eviction_ratio',
            aggfunc='mean'
        )
        
        report.append("```")
        report.append(str(eviction_table))
        report.append("```\n")
        
        # Add performance comparison
        report.append("## Performance Comparison (Duration in ms)")
        
        perf_table = self.results.pivot_table(
            index=['distribution'], 
            columns=['policy'], 
            values='duration_ms',
            aggfunc='mean'
        )
        
        report.append("```")
        report.append(str(perf_table))
        report.append("```\n")
        
        # Add best policy recommendations
        report.append("## Best Policy Recommendations")
        report.append("")
        
        # Find best policy for each distribution
        for dist in self.results['distribution'].unique():
            dist_data = self.results[self.results['distribution'] == dist]
            best_policy = dist_data.loc[dist_data['hit_ratio'].idxmax()]
            
            report.append(f"### For {dist} distribution:")
            report.append(f"- Best policy: **{best_policy['policy']}** with hit ratio of {best_policy['hit_ratio']:.2f}%")
            report.append(f"- Cache size: {best_policy['cache_size']} entries")
            report.append(f"- Eviction ratio: {best_policy['eviction_ratio']:.2f}%")
            report.append("")
        
        # Add conclusion
        report.append("## Conclusion")
        report.append("Based on the hit ratio test results, we can draw the following conclusions:")
        report.append("")
        
        # Add distribution-specific conclusions
        if 'uniform' in self.results['distribution'].values:
            uniform_data = self.results[self.results['distribution'] == 'uniform']
            best_uniform = uniform_data.loc[uniform_data['hit_ratio'].idxmax()]
            report.append(f"1. **Uniform Distribution**: {best_uniform['policy']} performs best with a hit ratio of {best_uniform['hit_ratio']:.2f}%.")
        
        if 'zipf-1.07' in self.results['distribution'].values:
            zipflow_data = self.results[self.results['distribution'] == 'zipf-1.07']
            best_zipflow = zipflow_data.loc[zipflow_data['hit_ratio'].idxmax()]
            report.append(f"2. **Low-skew Zipf Distribution**: {best_zipflow['policy']} performs best with a hit ratio of {best_zipflow['hit_ratio']:.2f}%.")
        
        if 'zipf-1.2' in self.results['distribution'].values:
            zipfhigh_data = self.results[self.results['distribution'] == 'zipf-1.2']
            best_zipfhigh = zipfhigh_data.loc[zipfhigh_data['hit_ratio'].idxmax()]
            report.append(f"3. **High-skew Zipf Distribution**: {best_zipfhigh['policy']} performs best with a hit ratio of {best_zipfhigh['hit_ratio']:.2f}%.")
        
        # Add general conclusions
        report.append("")
        report.append("### General Observations:")
        report.append("- LRU and LFU policies generally perform better for skewed access patterns (Zipf distributions).")
        report.append("- For uniform access patterns, the differences between policies are less pronounced.")
        report.append("- Larger cache sizes predictably lead to higher hit ratios across all policies and distributions.")
        
        # Write report to file
        report_content = "\n".join(report)
        report_file = os.path.join(self.report_dir, f'hitratio_report_{datetime.now().strftime("%Y%m%d")}.md')
        
        with open(report_file, 'w') as f:
            f.write(report_content)
        
        return report_file

def main():
    """Main function to run the hit ratio analyzer."""
    parser = argparse.ArgumentParser(description='Analyze cache hit ratio test results')
    parser.add_argument('--input', '-i', required=True, help='Directory containing hit ratio test result files')
    parser.add_argument('--output', '-o', required=True, help='Directory to save analysis results')
    parser.add_argument('--pattern', '-p', help='Regex pattern to filter input files')
    
    args = parser.parse_args()
    
    analyzer = HitRatioAnalyzer(args.input, args.output)
    
    print("Loading hit ratio test results...")
    results = analyzer.load_test_results(args.pattern)
    print(f"Loaded {len(results)} hit ratio test results")
    
    print("Generating descriptive statistics...")
    stats = analyzer.generate_descriptive_stats()
    
    print("Generating hit ratio comparison plots...")
    plot_files = analyzer.plot_hit_ratio_comparison()
    print(f"Generated {len(plot_files)} plot files")
    
    print("Generating summary report...")
    report_file = analyzer.generate_summary_report()
    print(f"Generated summary report: {report_file}")
    
    print("Analysis complete!")

if __name__ == "__main__":
    main() 