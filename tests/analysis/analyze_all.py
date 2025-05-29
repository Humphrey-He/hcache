#!/usr/bin/env python3
"""
HCache Comprehensive Analysis Tool

This script coordinates all analysis tools to generate a comprehensive report.
"""

import os
import sys
import subprocess
import argparse
import shutil
import datetime
import pandas as pd
import matplotlib.pyplot as plt
from pathlib import Path

# Import individual analyzers
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
try:
    from benchmark_analyzer import BenchmarkAnalyzer
    from hitratio_analyzer import HitRatioAnalyzer
    from concurrency_analyzer import ConcurrencyAnalyzer
    from pprof_analyzer import PprofAnalyzer
except ImportError as e:
    print(f"Error importing analyzer modules: {e}")
    print("Make sure all analyzer scripts are in the same directory as this script.")
    sys.exit(1)

class ComprehensiveAnalyzer:
    """Coordinates all analysis tools and generates a comprehensive report."""
    
    def __init__(self, base_dir, output_dir):
        """
        Initialize the analyzer with base directory and output directory.
        
        Args:
            base_dir: Base directory containing test results
            output_dir: Directory to save analysis results
        """
        self.base_dir = base_dir
        self.output_dir = output_dir
        
        # Define subdirectories for different test results
        self.benchmark_dir = os.path.join(base_dir, 'benchmark', 'result')
        self.hitratio_dir = os.path.join(base_dir, 'hitratio', 'result')
        self.concurrency_dir = os.path.join(base_dir, 'concurrency', 'result')
        self.pprof_dir = os.path.join(base_dir, 'pprof')
        
        # Define output subdirectories
        self.benchmark_output = os.path.join(output_dir, 'benchmark')
        self.hitratio_output = os.path.join(output_dir, 'hitratio')
        self.concurrency_output = os.path.join(output_dir, 'concurrency')
        self.pprof_output = os.path.join(output_dir, 'pprof')
        self.summary_output = os.path.join(output_dir, 'summary')
        
        # Create output directories
        for directory in [self.output_dir, self.benchmark_output, self.hitratio_output, 
                         self.concurrency_output, self.pprof_output, self.summary_output]:
            os.makedirs(directory, exist_ok=True)
        
        # Initialize analyzer instances
        self.benchmark_analyzer = None
        self.hitratio_analyzer = None
        self.concurrency_analyzer = None
        self.pprof_analyzer = None
        
        # Store report paths
        self.report_paths = {
            'benchmark': None,
            'hitratio': None,
            'concurrency': None,
            'pprof': None,
            'summary': None
        }
    
    def run_benchmark_analysis(self):
        """
        Run benchmark analysis.
        
        Returns:
            Path to the benchmark report
        """
        print("\n=== Running Benchmark Analysis ===")
        
        if not os.path.isdir(self.benchmark_dir):
            print(f"Benchmark results directory not found: {self.benchmark_dir}")
            return None
        
        try:
            self.benchmark_analyzer = BenchmarkAnalyzer(self.benchmark_dir, self.benchmark_output)
            
            print("Loading benchmark results...")
            results = self.benchmark_analyzer.load_benchmark_results()
            print(f"Loaded {len(results)} benchmark results")
            
            print("Generating descriptive statistics...")
            stats, percentiles = self.benchmark_analyzer.generate_descriptive_stats()
            
            print("Generating performance comparison plots...")
            plot_files = self.benchmark_analyzer.plot_performance_comparison()
            print(f"Generated {len(plot_files)} plot files")
            
            print("Generating summary report...")
            report_file = self.benchmark_analyzer.generate_summary_report()
            print(f"Generated benchmark report: {report_file}")
            
            self.report_paths['benchmark'] = report_file
            return report_file
        
        except Exception as e:
            print(f"Error running benchmark analysis: {e}")
            return None
    
    def run_hitratio_analysis(self):
        """
        Run hit ratio analysis.
        
        Returns:
            Path to the hit ratio report
        """
        print("\n=== Running Hit Ratio Analysis ===")
        
        if not os.path.isdir(self.hitratio_dir):
            print(f"Hit ratio results directory not found: {self.hitratio_dir}")
            return None
        
        try:
            self.hitratio_analyzer = HitRatioAnalyzer(self.hitratio_dir, self.hitratio_output)
            
            print("Loading hit ratio test results...")
            results = self.hitratio_analyzer.load_test_results()
            print(f"Loaded {len(results)} hit ratio test results")
            
            print("Generating descriptive statistics...")
            stats = self.hitratio_analyzer.generate_descriptive_stats()
            
            print("Generating hit ratio comparison plots...")
            plot_files = self.hitratio_analyzer.plot_hit_ratio_comparison()
            print(f"Generated {len(plot_files)} plot files")
            
            print("Generating summary report...")
            report_file = self.hitratio_analyzer.generate_summary_report()
            print(f"Generated hit ratio report: {report_file}")
            
            self.report_paths['hitratio'] = report_file
            return report_file
        
        except Exception as e:
            print(f"Error running hit ratio analysis: {e}")
            return None
    
    def run_concurrency_analysis(self):
        """
        Run concurrency analysis.
        
        Returns:
            Path to the concurrency report
        """
        print("\n=== Running Concurrency Analysis ===")
        
        if not os.path.isdir(self.concurrency_dir):
            print(f"Concurrency results directory not found: {self.concurrency_dir}")
            return None
        
        try:
            self.concurrency_analyzer = ConcurrencyAnalyzer(self.concurrency_dir, self.concurrency_output)
            
            print("Loading concurrency test results...")
            results = self.concurrency_analyzer.load_concurrency_results()
            print(f"Loaded {len(results)} concurrency test results")
            
            print("Generating descriptive statistics...")
            stats = self.concurrency_analyzer.generate_descriptive_stats()
            
            print("Generating concurrency test plots...")
            plot_files = self.concurrency_analyzer.plot_concurrency_results()
            print(f"Generated {len(plot_files)} plot files")
            
            print("Generating summary report...")
            report_file = self.concurrency_analyzer.generate_summary_report()
            print(f"Generated concurrency report: {report_file}")
            
            self.report_paths['concurrency'] = report_file
            return report_file
        
        except Exception as e:
            print(f"Error running concurrency analysis: {e}")
            return None
    
    def run_pprof_analysis(self):
        """
        Run pprof analysis.
        
        Returns:
            Path to the pprof report
        """
        print("\n=== Running pprof Analysis ===")
        
        if not os.path.isdir(self.pprof_dir):
            print(f"pprof results directory not found: {self.pprof_dir}")
            return None
        
        try:
            self.pprof_analyzer = PprofAnalyzer(self.pprof_dir, self.pprof_output)
            
            print("Analyzing pprof profiles...")
            results = self.pprof_analyzer.analyze_profiles()
            print(f"Analyzed {len(results['profiles'])} pprof profiles")
            
            print("Generating top functions plots...")
            plot_files = self.pprof_analyzer.generate_top_functions_plots()
            print(f"Generated {len(plot_files)} plot files")
            
            print("Generating summary report...")
            report_file = self.pprof_analyzer.generate_summary_report()
            print(f"Generated pprof report: {report_file}")
            
            self.report_paths['pprof'] = report_file
            return report_file
        
        except Exception as e:
            print(f"Error running pprof analysis: {e}")
            return None
    
    def generate_comprehensive_report(self):
        """
        Generate a comprehensive summary report combining all analysis results.
        
        Returns:
            Path to the comprehensive report
        """
        print("\n=== Generating Comprehensive Report ===")
        
        # Generate timestamp
        timestamp = datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        
        # Start building the report
        report = [
            "# HCache Comprehensive Analysis Report",
            f"Generated on: {timestamp}\n",
            "## Overview",
            "This report combines the results from multiple analysis tools to provide a comprehensive view of HCache performance.\n"
        ]
        
        # Add benchmark analysis summary
        report.append("## Benchmark Analysis")
        if self.report_paths['benchmark']:
            benchmark_report_path = Path(self.report_paths['benchmark'])
            try:
                with open(benchmark_report_path, 'r') as f:
                    content = f.read()
                    
                    # Extract summary and conclusion sections
                    summary_section = extract_section(content, "Summary Statistics", "Performance Comparison")
                    conclusion_section = extract_section(content, "Conclusion", None)
                    
                    if summary_section:
                        report.append(summary_section)
                    
                    if conclusion_section:
                        report.append(conclusion_section)
                    
                report.append(f"[View Full Benchmark Report](../benchmark/reports/{benchmark_report_path.name})\n")
            except Exception as e:
                report.append(f"Error extracting benchmark report content: {e}\n")
        else:
            report.append("No benchmark analysis results available.\n")
        
        # Add hit ratio analysis summary
        report.append("## Hit Ratio Analysis")
        if self.report_paths['hitratio']:
            hitratio_report_path = Path(self.report_paths['hitratio'])
            try:
                with open(hitratio_report_path, 'r') as f:
                    content = f.read()
                    
                    # Extract best policy recommendations and conclusion sections
                    recommendations_section = extract_section(content, "Best Policy Recommendations", "Conclusion")
                    conclusion_section = extract_section(content, "Conclusion", None)
                    
                    if recommendations_section:
                        report.append(recommendations_section)
                    
                    if conclusion_section:
                        report.append(conclusion_section)
                    
                report.append(f"[View Full Hit Ratio Report](../hitratio/reports/{hitratio_report_path.name})\n")
            except Exception as e:
                report.append(f"Error extracting hit ratio report content: {e}\n")
        else:
            report.append("No hit ratio analysis results available.\n")
        
        # Add concurrency analysis summary
        report.append("## Concurrency Analysis")
        if self.report_paths['concurrency']:
            concurrency_report_path = Path(self.report_paths['concurrency'])
            try:
                with open(concurrency_report_path, 'r') as f:
                    content = f.read()
                    
                    # Extract optimal concurrency level and conclusion sections
                    optimal_section = extract_section(content, "Optimal Concurrency Level", "Conclusion")
                    conclusion_section = extract_section(content, "Conclusion", None)
                    
                    if optimal_section:
                        report.append(optimal_section)
                    
                    if conclusion_section:
                        report.append(conclusion_section)
                    
                report.append(f"[View Full Concurrency Report](../concurrency/reports/{concurrency_report_path.name})\n")
            except Exception as e:
                report.append(f"Error extracting concurrency report content: {e}\n")
        else:
            report.append("No concurrency analysis results available.\n")
        
        # Add pprof analysis summary
        report.append("## Performance Profiling Analysis")
        if self.report_paths['pprof']:
            pprof_report_path = Path(self.report_paths['pprof'])
            try:
                with open(pprof_report_path, 'r') as f:
                    content = f.read()
                    
                    # Extract analysis and recommendations section
                    recommendations_section = extract_section(content, "Analysis and Recommendations", None)
                    
                    if recommendations_section:
                        report.append(recommendations_section)
                    
                report.append(f"[View Full Profiling Report](../pprof/reports/{pprof_report_path.name})\n")
            except Exception as e:
                report.append(f"Error extracting pprof report content: {e}\n")
        else:
            report.append("No performance profiling results available.\n")
        
        # Add comprehensive conclusion
        report.append("## Comprehensive Conclusion")
        report.append("Based on the combined analysis results, we can draw the following conclusions about HCache performance:")
        report.append("")
        
        # Add benchmark conclusions
        if self.benchmark_analyzer and self.benchmark_analyzer.results is not None:
            report.append("### Performance Characteristics")
            report.append("- **Latency**: HCache demonstrates [low/medium/high] latency across various operations.")
            report.append("- **Memory Efficiency**: Memory allocation patterns show [efficient/inefficient] usage.")
            report.append("- **Scalability**: Performance [scales well/degrades] with increasing data sizes.")
            report.append("")
        
        # Add hit ratio conclusions
        if self.hitratio_analyzer and self.hitratio_analyzer.results is not None:
            report.append("### Cache Effectiveness")
            
            # Try to determine best policy
            try:
                best_policies = {}
                for dist in self.hitratio_analyzer.results['distribution'].unique():
                    dist_data = self.hitratio_analyzer.results[self.hitratio_analyzer.results['distribution'] == dist]
                    best_policy = dist_data.loc[dist_data['hit_ratio'].idxmax()]
                    best_policies[dist] = best_policy['policy']
                
                report.append("- **Best Eviction Policies**:")
                for dist, policy in best_policies.items():
                    report.append(f"  - For {dist} distribution: **{policy}**")
            except:
                pass
            
            report.append("- **Hit Ratio Optimization**: The cache hit ratio can be optimized by selecting appropriate eviction policies for different access patterns.")
            report.append("- **Cache Sizing**: Larger cache sizes predictably lead to higher hit ratios, with diminishing returns beyond certain thresholds.")
            report.append("")
        
        # Add concurrency conclusions
        if self.concurrency_analyzer and self.concurrency_analyzer.results is not None:
            report.append("### Concurrency Performance")
            report.append("- **Optimal Concurrency**: HCache performs best at [specific concurrency level] concurrent operations.")
            report.append("- **Throughput**: Maximum throughput is achieved at [specific concurrency level] with [throughput value] requests per second.")
            report.append("- **Latency Under Load**: Latency remains [stable/increases] as concurrency increases, indicating [good/poor] scalability.")
            report.append("")
        
        # Add pprof conclusions
        if self.pprof_analyzer and self.pprof_analyzer.results is not None:
            report.append("### Performance Bottlenecks")
            report.append("- **CPU Hotspots**: The most CPU-intensive operations are in [specific functions/areas].")
            report.append("- **Memory Allocation**: Memory allocation is concentrated in [specific functions/areas].")
            report.append("- **Optimization Opportunities**: Performance could be improved by optimizing [specific areas].")
            report.append("")
        
        # Add final recommendations
        report.append("### Recommendations for Improvement")
        report.append("1. **Eviction Policy**: Use [specific policy] for general-purpose caching, and consider adaptive policies for mixed workloads.")
        report.append("2. **Concurrency Tuning**: Configure the cache with [specific concurrency settings] for optimal performance.")
        report.append("3. **Memory Optimization**: Reduce memory allocations in [specific areas] to improve GC behavior.")
        report.append("4. **Algorithm Improvements**: Consider alternative implementations for [specific operations] to reduce CPU usage.")
        report.append("5. **Benchmarking**: Regularly benchmark with realistic workloads to ensure performance remains optimal.")
        
        # Write report to file
        report_content = "\n".join(report)
        report_file = os.path.join(self.summary_output, f'comprehensive_report_{datetime.datetime.now().strftime("%Y%m%d")}.md')
        
        with open(report_file, 'w') as f:
            f.write(report_content)
        
        # Create an HTML version
        html_report = f"""
        <!DOCTYPE html>
        <html>
        <head>
            <title>HCache Comprehensive Analysis Report</title>
            <style>
                body {{ font-family: Arial, sans-serif; margin: 20px; line-height: 1.6; }}
                h1, h2, h3 {{ color: #333; }}
                table {{ border-collapse: collapse; width: 100%; margin-bottom: 20px; }}
                th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
                th {{ background-color: #f2f2f2; }}
                tr:nth-child(even) {{ background-color: #f9f9f9; }}
                .container {{ max-width: 1200px; margin: 0 auto; }}
                img {{ max-width: 100%; }}
                a {{ color: #0366d6; text-decoration: none; }}
                a:hover {{ text-decoration: underline; }}
                pre {{ background-color: #f6f8fa; padding: 16px; overflow: auto; line-height: 1.45; border-radius: 3px; }}
                code {{ font-family: SFMono-Regular, Consolas, Liberation Mono, Menlo, monospace; }}
            </style>
        </head>
        <body>
            <div class="container">
                <h1>HCache Comprehensive Analysis Report</h1>
                <p>Generated on: {timestamp}</p>
                
                {markdown_to_html(report_content)}
            </div>
        </body>
        </html>
        """
        
        html_report_file = os.path.join(self.summary_output, f'comprehensive_report_{datetime.datetime.now().strftime("%Y%m%d")}.html')
        with open(html_report_file, 'w') as f:
            f.write(html_report)
        
        # Create Excel summary
        self.create_excel_summary()
        
        self.report_paths['summary'] = report_file
        print(f"Generated comprehensive report: {report_file}")
        print(f"Generated HTML report: {html_report_file}")
        
        return report_file
    
    def create_excel_summary(self):
        """Create an Excel summary with key metrics from all analyses."""
        excel_file = os.path.join(self.summary_output, f'hcache_metrics_summary_{datetime.datetime.now().strftime("%Y%m%d")}.xlsx')
        
        with pd.ExcelWriter(excel_file, engine='openpyxl') as writer:
            # Benchmark summary
            if self.benchmark_analyzer and self.benchmark_analyzer.results is not None:
                # Create a summary of benchmark results
                benchmark_summary = self.benchmark_analyzer.results.pivot_table(
                    index=['name'], 
                    columns=['ValueSize'], 
                    values=['ns_per_op', 'bytes_per_op', 'allocs_per_op'],
                    aggfunc='mean'
                )
                benchmark_summary.to_excel(writer, sheet_name='Benchmark')
            
            # Hit ratio summary
            if self.hitratio_analyzer and self.hitratio_analyzer.results is not None:
                # Create a summary of hit ratio results
                hitratio_summary = self.hitratio_analyzer.results.pivot_table(
                    index=['policy'], 
                    columns=['distribution'], 
                    values=['hit_ratio'],
                    aggfunc='mean'
                )
                hitratio_summary.to_excel(writer, sheet_name='HitRatio')
            
            # Concurrency summary
            if self.concurrency_analyzer and self.concurrency_analyzer.results is not None:
                # Create a summary of concurrency results
                concurrency_summary = self.concurrency_analyzer.results.pivot_table(
                    index=['cache_config'], 
                    columns=['concurrency'], 
                    values=['latency_mean', 'throughput', 'success_rate'],
                    aggfunc='mean'
                )
                concurrency_summary.to_excel(writer, sheet_name='Concurrency')
        
        print(f"Generated Excel summary: {excel_file}")
        return excel_file

def extract_section(content, start_section, end_section=None):
    """
    Extract a section from markdown content.
    
    Args:
        content: Markdown content
        start_section: Section title to start extraction from
        end_section: Section title to end extraction at (optional)
        
    Returns:
        Extracted section content
    """
    lines = content.split('\n')
    
    # Find start section
    start_idx = -1
    for i, line in enumerate(lines):
        if line.startswith(f'## {start_section}') or line.startswith(f'### {start_section}'):
            start_idx = i
            break
    
    if start_idx == -1:
        return None
    
    # Find end section
    end_idx = len(lines)
    if end_section:
        for i in range(start_idx + 1, len(lines)):
            if lines[i].startswith(f'## {end_section}') or lines[i].startswith(f'### {end_section}'):
                end_idx = i
                break
    
    # Extract section
    section = lines[start_idx:end_idx]
    return '\n'.join(section)

def markdown_to_html(markdown_content):
    """
    Convert markdown to HTML (very simple conversion).
    
    Args:
        markdown_content: Markdown content
        
    Returns:
        HTML content
    """
    # This is a very simple conversion, not a full markdown parser
    html = markdown_content
    
    # Convert headers
    html = html.replace('# ', '<h1>').replace('\n# ', '\n<h1>')
    html = html.replace('## ', '<h2>').replace('\n## ', '\n<h2>')
    html = html.replace('### ', '<h3>').replace('\n### ', '\n<h3>')
    html = html.replace('#### ', '<h4>').replace('\n#### ', '\n<h4>')
    
    # Close headers
    html = html.replace('\n<h1>', '</h1>\n<h1>')
    html = html.replace('\n<h2>', '</h2>\n<h2>')
    html = html.replace('\n<h3>', '</h3>\n<h3>')
    html = html.replace('\n<h4>', '</h4>\n<h4>')
    
    # Add closing tags for the last headers
    if '<h1>' in html:
        html = html + '</h1>'
    if '<h2>' in html:
        html = html + '</h2>'
    if '<h3>' in html:
        html = html + '</h3>'
    if '<h4>' in html:
        html = html + '</h4>'
    
    # Convert links
    html = html.replace('[', '<a href="').replace(']', '</a>')
    html = html.replace('(', '">').replace(')', '')
    
    # Convert code blocks
    html = html.replace('```', '<pre><code>')
    html = html.replace('\n```', '</code></pre>')
    
    # Convert lists
    html = html.replace('\n- ', '\n<li>').replace('\n  - ', '\n  <li>')
    html = html.replace('\n1. ', '\n<li>').replace('\n  1. ', '\n  <li>')
    
    # Convert paragraphs
    paragraphs = html.split('\n\n')
    for i, p in enumerate(paragraphs):
        if not p.startswith('<h') and not p.startswith('<pre') and not p.startswith('<li'):
            paragraphs[i] = f'<p>{p}</p>'
    
    html = '\n\n'.join(paragraphs)
    
    return html

def main():
    """Main function to run the comprehensive analyzer."""
    parser = argparse.ArgumentParser(description='Run comprehensive analysis on HCache test results')
    parser.add_argument('--base-dir', '-b', default='..', help='Base directory containing test results')
    parser.add_argument('--output', '-o', default='./output', help='Directory to save analysis results')
    parser.add_argument('--skip-benchmark', action='store_true', help='Skip benchmark analysis')
    parser.add_argument('--skip-hitratio', action='store_true', help='Skip hit ratio analysis')
    parser.add_argument('--skip-concurrency', action='store_true', help='Skip concurrency analysis')
    parser.add_argument('--skip-pprof', action='store_true', help='Skip pprof analysis')
    
    args = parser.parse_args()
    
    analyzer = ComprehensiveAnalyzer(args.base_dir, args.output)
    
    # Run individual analyses
    if not args.skip_benchmark:
        analyzer.run_benchmark_analysis()
    
    if not args.skip_hitratio:
        analyzer.run_hitratio_analysis()
    
    if not args.skip_concurrency:
        analyzer.run_concurrency_analysis()
    
    if not args.skip_pprof:
        analyzer.run_pprof_analysis()
    
    # Generate comprehensive report
    analyzer.generate_comprehensive_report()
    
    print("\nAnalysis complete! Results saved to:", args.output)

if __name__ == "__main__":
    main() 