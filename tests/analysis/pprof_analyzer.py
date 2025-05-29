#!/usr/bin/env python3
"""
HCache pprof Analysis Tool

This script processes Go pprof profiles and generates visualizations and analysis.
"""

import os
import re
import subprocess
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
import plotly.express as px
import plotly.graph_objects as go
from plotly.subplots import make_subplots
from datetime import datetime
import argparse
import json
import shutil

# Set style for matplotlib
plt.style.use('ggplot')
sns.set_theme(style="whitegrid")

class PprofAnalyzer:
    """Analyzes Go pprof profiles and generates visualizations."""
    
    def __init__(self, input_dir, output_dir):
        """
        Initialize the analyzer with input and output directories.
        
        Args:
            input_dir: Directory containing pprof profile files
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
        self.flame_dir = os.path.join(output_dir, 'flamegraphs')
        
        for directory in [self.img_dir, self.csv_dir, self.html_dir, self.report_dir, self.flame_dir]:
            os.makedirs(directory, exist_ok=True)
    
    def check_go_tool_pprof(self):
        """
        Check if go tool pprof is available.
        
        Returns:
            bool: True if go tool pprof is available, False otherwise
        """
        try:
            subprocess.run(['go', 'tool', 'pprof', '--help'], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            return True
        except (subprocess.SubprocessError, FileNotFoundError):
            return False
    
    def extract_profile_metadata(self, filename):
        """
        Extract metadata from a pprof profile filename.
        
        Args:
            filename: Profile filename
            
        Returns:
            dict: Profile metadata
        """
        metadata = {
            'profile_type': 'unknown',
            'test_name': 'unknown',
            'date': '00000000',
            'time': '000000'
        }
        
        # Extract profile type
        if 'cpu' in filename.lower():
            metadata['profile_type'] = 'cpu'
        elif 'heap' in filename.lower() or 'mem' in filename.lower():
            metadata['profile_type'] = 'heap'
        elif 'block' in filename.lower():
            metadata['profile_type'] = 'block'
        elif 'mutex' in filename.lower():
            metadata['profile_type'] = 'mutex'
        elif 'goroutine' in filename.lower():
            metadata['profile_type'] = 'goroutine'
        
        # Extract test name
        test_match = re.search(r'([a-zA-Z]+)_\d{8}', filename)
        if test_match:
            metadata['test_name'] = test_match.group(1)
        
        # Extract date and time
        date_match = re.search(r'(\d{8})_?(\d{6})?', filename)
        if date_match:
            metadata['date'] = date_match.group(1)
            if date_match.group(2):
                metadata['time'] = date_match.group(2)
        
        return metadata
    
    def generate_flamegraph(self, profile_path, output_path):
        """
        Generate a flamegraph from a pprof profile.
        
        Args:
            profile_path: Path to the pprof profile
            output_path: Path to save the flamegraph
            
        Returns:
            bool: True if successful, False otherwise
        """
        if not self.check_go_tool_pprof():
            print("Warning: go tool pprof not found, skipping flamegraph generation")
            return False
        
        try:
            # Generate SVG flamegraph
            svg_path = output_path + '.svg'
            subprocess.run([
                'go', 'tool', 'pprof', 
                '-flamegraph', 
                '-output', svg_path,
                profile_path
            ], stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)
            
            # Generate HTML with interactive flamegraph
            html_path = output_path + '.html'
            with open(html_path, 'w') as f:
                f.write(f"""
                <!DOCTYPE html>
                <html>
                <head>
                    <title>Flamegraph: {os.path.basename(profile_path)}</title>
                    <style>
                        body {{ font-family: Arial, sans-serif; margin: 20px; }}
                        h1 {{ color: #333; }}
                        .container {{ max-width: 1200px; margin: 0 auto; }}
                    </style>
                </head>
                <body>
                    <div class="container">
                        <h1>Flamegraph: {os.path.basename(profile_path)}</h1>
                        <div>
                            <embed src="{os.path.basename(svg_path)}" type="image/svg+xml" width="100%" height="800px" />
                        </div>
                    </div>
                </body>
                </html>
                """)
            
            return True
        except subprocess.SubprocessError as e:
            print(f"Error generating flamegraph: {e}")
            return False
    
    def extract_top_functions(self, profile_path, n=20):
        """
        Extract the top N functions from a pprof profile.
        
        Args:
            profile_path: Path to the pprof profile
            n: Number of top functions to extract
            
        Returns:
            DataFrame: Top functions with their metrics
        """
        if not self.check_go_tool_pprof():
            print("Warning: go tool pprof not found, skipping top functions extraction")
            return pd.DataFrame()
        
        try:
            # Run pprof to get top functions in text format
            result = subprocess.run([
                'go', 'tool', 'pprof', 
                '-top', 
                '-nodecount', str(n),
                profile_path
            ], stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True, text=True)
            
            # Parse the output
            lines = result.stdout.strip().split('\n')
            
            # Find the header line
            header_idx = -1
            for i, line in enumerate(lines):
                if 'flat' in line and 'cum' in line and '%' in line:
                    header_idx = i
                    break
            
            if header_idx == -1 or header_idx + 1 >= len(lines):
                return pd.DataFrame()
            
            # Parse the data lines
            data = []
            for line in lines[header_idx+1:]:
                if not line.strip():
                    continue
                
                # Split the line by whitespace, but keep function name intact
                parts = line.strip().split()
                if len(parts) < 5:
                    continue
                
                # The function name might contain spaces, so join the remaining parts
                flat_pct = float(parts[0].replace('%', ''))
                flat_val = parts[1]
                cum_pct = float(parts[2].replace('%', ''))
                cum_val = parts[3]
                func_name = ' '.join(parts[4:])
                
                data.append({
                    'flat_pct': flat_pct,
                    'flat_val': flat_val,
                    'cum_pct': cum_pct,
                    'cum_val': cum_val,
                    'function': func_name
                })
            
            return pd.DataFrame(data)
        except subprocess.SubprocessError as e:
            print(f"Error extracting top functions: {e}")
            return pd.DataFrame()
    
    def analyze_profiles(self):
        """
        Analyze all pprof profiles in the input directory.
        
        Returns:
            dict: Analysis results
        """
        if not self.check_go_tool_pprof():
            print("Warning: go tool pprof not found, analysis will be limited")
        
        results = {
            'profiles': [],
            'top_functions': {}
        }
        
        # Find all pprof profiles
        profile_files = []
        for filename in os.listdir(self.input_dir):
            if filename.endswith('.pprof') or filename.endswith('.pb.gz'):
                profile_files.append(os.path.join(self.input_dir, filename))
        
        if not profile_files:
            print(f"No pprof profiles found in {self.input_dir}")
            return results
        
        # Process each profile
        for profile_path in profile_files:
            basename = os.path.basename(profile_path)
            metadata = self.extract_profile_metadata(basename)
            
            profile_result = {
                'filename': basename,
                'path': profile_path,
                'metadata': metadata,
                'flamegraph_path': None,
                'top_functions': None
            }
            
            # Generate flamegraph
            flamegraph_basename = os.path.splitext(basename)[0]
            flamegraph_path = os.path.join(self.flame_dir, flamegraph_basename)
            if self.generate_flamegraph(profile_path, flamegraph_path):
                profile_result['flamegraph_path'] = flamegraph_path + '.svg'
            
            # Extract top functions
            top_functions = self.extract_top_functions(profile_path)
            if not top_functions.empty:
                profile_result['top_functions'] = top_functions.to_dict('records')
                results['top_functions'][basename] = top_functions
            
            results['profiles'].append(profile_result)
        
        self.results = results
        return results
    
    def generate_top_functions_plots(self):
        """
        Generate plots for top functions.
        
        Returns:
            list: Paths to generated plot files
        """
        if not self.results or not self.results.get('top_functions'):
            return []
        
        plot_files = []
        
        # Process each profile's top functions
        for profile_name, top_functions in self.results['top_functions'].items():
            if top_functions.empty:
                continue
            
            # Create a horizontal bar chart of top functions by flat percentage
            plt.figure(figsize=(12, 10))
            top_n = min(10, len(top_functions))
            top_flat = top_functions.nlargest(top_n, 'flat_pct')
            
            # Clean function names for better display
            top_flat['function_short'] = top_flat['function'].apply(
                lambda x: re.sub(r'^.*/', '', x)  # Remove package path
            )
            
            sns.barplot(y='function_short', x='flat_pct', data=top_flat, palette='viridis')
            plt.title(f'Top {top_n} Functions by Flat Percentage - {profile_name}')
            plt.xlabel('Flat Percentage (%)')
            plt.ylabel('Function')
            plt.tight_layout()
            
            # Save plot
            plot_file = os.path.join(self.img_dir, f'{os.path.splitext(profile_name)[0]}_top_flat.png')
            plt.savefig(plot_file, dpi=300, bbox_inches='tight')
            plt.close()
            plot_files.append(plot_file)
            
            # Create a horizontal bar chart of top functions by cumulative percentage
            plt.figure(figsize=(12, 10))
            top_cum = top_functions.nlargest(top_n, 'cum_pct')
            
            # Clean function names for better display
            top_cum['function_short'] = top_cum['function'].apply(
                lambda x: re.sub(r'^.*/', '', x)  # Remove package path
            )
            
            sns.barplot(y='function_short', x='cum_pct', data=top_cum, palette='magma')
            plt.title(f'Top {top_n} Functions by Cumulative Percentage - {profile_name}')
            plt.xlabel('Cumulative Percentage (%)')
            plt.ylabel('Function')
            plt.tight_layout()
            
            # Save plot
            plot_file = os.path.join(self.img_dir, f'{os.path.splitext(profile_name)[0]}_top_cum.png')
            plt.savefig(plot_file, dpi=300, bbox_inches='tight')
            plt.close()
            plot_files.append(plot_file)
            
            # Save top functions to CSV
            csv_file = os.path.join(self.csv_dir, f'{os.path.splitext(profile_name)[0]}_top_functions.csv')
            top_functions.to_csv(csv_file, index=False)
            
            # Create interactive bar chart with Plotly
            fig = px.bar(
                top_flat, 
                y='function_short', 
                x='flat_pct',
                orientation='h',
                title=f'Top {top_n} Functions by Flat Percentage - {profile_name}',
                labels={'function_short': 'Function', 'flat_pct': 'Flat Percentage (%)'},
                color='flat_pct',
                color_continuous_scale='viridis'
            )
            
            html_file = os.path.join(self.html_dir, f'{os.path.splitext(profile_name)[0]}_top_flat.html')
            fig.write_html(html_file)
            plot_files.append(html_file)
        
        return plot_files
    
    def generate_summary_report(self):
        """
        Generate a comprehensive summary report in markdown format.
        
        Returns:
            str: Path to the generated report file
        """
        if not self.results:
            raise ValueError("No pprof analysis results available")
        
        # Generate timestamp
        timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        
        # Start building the report
        report = [
            "# HCache pprof Analysis Report",
            f"Generated on: {timestamp}\n",
            f"Analyzed {len(self.results['profiles'])} pprof profiles\n",
            "## Profile Summary",
        ]
        
        # Add profile summary table
        report.append("| Profile | Type | Test | Date | Flamegraph |")
        report.append("|---------|------|------|------|-----------|")
        
        for profile in self.results['profiles']:
            metadata = profile['metadata']
            flamegraph_link = f"[View]({os.path.basename(profile['flamegraph_path'])})" if profile['flamegraph_path'] else "N/A"
            report.append(f"| {profile['filename']} | {metadata['profile_type']} | {metadata['test_name']} | {metadata['date']} | {flamegraph_link} |")
        
        report.append("")
        
        # Add top functions for each profile
        report.append("## Top Functions Analysis")
        
        for profile in self.results['profiles']:
            if not profile.get('top_functions'):
                continue
            
            report.append(f"\n### {profile['filename']}")
            report.append("")
            
            # Add top 5 functions table
            report.append("#### Top 5 Functions by Flat Percentage")
            report.append("| Function | Flat % | Flat | Cum % | Cum |")
            report.append("|----------|--------|------|-------|-----|")
            
            top_functions = pd.DataFrame(profile['top_functions'])
            top_5 = top_functions.nlargest(5, 'flat_pct')
            
            for _, row in top_5.iterrows():
                report.append(f"| {row['function']} | {row['flat_pct']} | {row['flat_val']} | {row['cum_pct']} | {row['cum_val']} |")
            
            report.append("")
            
            # Add links to plots
            basename = os.path.splitext(profile['filename'])[0]
            report.append(f"[View Top Functions by Flat Percentage](../images/{basename}_top_flat.png)")
            report.append(f"[View Top Functions by Cumulative Percentage](../images/{basename}_top_cum.png)")
            report.append(f"[View Interactive Chart](../html/{basename}_top_flat.html)")
            
            if profile['flamegraph_path']:
                report.append(f"[View Flamegraph](../flamegraphs/{os.path.basename(profile['flamegraph_path'])})")
            
            report.append("")
        
        # Add analysis and recommendations
        report.append("## Analysis and Recommendations")
        report.append("")
        
        # CPU profile analysis
        cpu_profiles = [p for p in self.results['profiles'] if p['metadata']['profile_type'] == 'cpu']
        if cpu_profiles:
            report.append("### CPU Profile Analysis")
            report.append("")
            report.append("The CPU profiles show the following hotspots:")
            report.append("")
            
            for profile in cpu_profiles:
                if not profile.get('top_functions'):
                    continue
                
                top_functions = pd.DataFrame(profile['top_functions'])
                top_3 = top_functions.nlargest(3, 'flat_pct')
                
                report.append(f"**{profile['filename']}**:")
                for _, row in top_3.iterrows():
                    report.append(f"- {row['function']}: {row['flat_pct']}% ({row['flat_val']})")
                report.append("")
            
            report.append("**Recommendations:**")
            report.append("- Consider optimizing the most time-consuming functions identified above.")
            report.append("- Look for opportunities to reduce allocations in hot code paths.")
            report.append("- Consider using more efficient algorithms or data structures for critical operations.")
            report.append("")
        
        # Heap profile analysis
        heap_profiles = [p for p in self.results['profiles'] if p['metadata']['profile_type'] == 'heap']
        if heap_profiles:
            report.append("### Memory Profile Analysis")
            report.append("")
            report.append("The heap profiles show the following memory allocation hotspots:")
            report.append("")
            
            for profile in heap_profiles:
                if not profile.get('top_functions'):
                    continue
                
                top_functions = pd.DataFrame(profile['top_functions'])
                top_3 = top_functions.nlargest(3, 'flat_pct')
                
                report.append(f"**{profile['filename']}**:")
                for _, row in top_3.iterrows():
                    report.append(f"- {row['function']}: {row['flat_pct']}% ({row['flat_val']})")
                report.append("")
            
            report.append("**Recommendations:**")
            report.append("- Review memory allocation patterns in the functions identified above.")
            report.append("- Consider using object pools for frequently allocated objects.")
            report.append("- Look for opportunities to reduce garbage collection pressure by minimizing allocations.")
            report.append("- Consider using value types instead of pointers where appropriate.")
            report.append("")
        
        # Write report to file
        report_content = "\n".join(report)
        report_file = os.path.join(self.report_dir, f'pprof_report_{datetime.now().strftime("%Y%m%d")}.md')
        
        with open(report_file, 'w') as f:
            f.write(report_content)
        
        # Create an HTML version of the report
        html_report = f"""
        <!DOCTYPE html>
        <html>
        <head>
            <title>HCache pprof Analysis Report</title>
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
            </style>
        </head>
        <body>
            <div class="container">
                <h1>HCache pprof Analysis Report</h1>
                <p>Generated on: {timestamp}</p>
                <p>Analyzed {len(self.results['profiles'])} pprof profiles</p>
                
                <h2>Profile Summary</h2>
                <table>
                    <tr>
                        <th>Profile</th>
                        <th>Type</th>
                        <th>Test</th>
                        <th>Date</th>
                        <th>Flamegraph</th>
                    </tr>
        """
        
        for profile in self.results['profiles']:
            metadata = profile['metadata']
            flamegraph_link = f'<a href="../flamegraphs/{os.path.basename(profile["flamegraph_path"])}" target="_blank">View</a>' if profile['flamegraph_path'] else "N/A"
            html_report += f"""
                    <tr>
                        <td>{profile['filename']}</td>
                        <td>{metadata['profile_type']}</td>
                        <td>{metadata['test_name']}</td>
                        <td>{metadata['date']}</td>
                        <td>{flamegraph_link}</td>
                    </tr>
            """
        
        html_report += """
                </table>
                
                <h2>Top Functions Analysis</h2>
        """
        
        for profile in self.results['profiles']:
            if not profile.get('top_functions'):
                continue
            
            basename = os.path.splitext(profile['filename'])[0]
            html_report += f"""
                <h3>{profile['filename']}</h3>
                
                <h4>Top 5 Functions by Flat Percentage</h4>
                <table>
                    <tr>
                        <th>Function</th>
                        <th>Flat %</th>
                        <th>Flat</th>
                        <th>Cum %</th>
                        <th>Cum</th>
                    </tr>
            """
            
            top_functions = pd.DataFrame(profile['top_functions'])
            top_5 = top_functions.nlargest(5, 'flat_pct')
            
            for _, row in top_5.iterrows():
                html_report += f"""
                    <tr>
                        <td>{row['function']}</td>
                        <td>{row['flat_pct']}</td>
                        <td>{row['flat_val']}</td>
                        <td>{row['cum_pct']}</td>
                        <td>{row['cum_val']}</td>
                    </tr>
                """
            
            html_report += f"""
                </table>
                
                <p>
                    <a href="../images/{basename}_top_flat.png" target="_blank">View Top Functions by Flat Percentage</a><br>
                    <a href="../images/{basename}_top_cum.png" target="_blank">View Top Functions by Cumulative Percentage</a><br>
                    <a href="../html/{basename}_top_flat.html" target="_blank">View Interactive Chart</a>
            """
            
            if profile['flamegraph_path']:
                html_report += f"""
                    <br><a href="../flamegraphs/{os.path.basename(profile['flamegraph_path'])}" target="_blank">View Flamegraph</a>
                """
            
            html_report += "</p>"
        
        html_report += """
                <h2>Analysis and Recommendations</h2>
        """
        
        # CPU profile analysis
        if cpu_profiles:
            html_report += """
                <h3>CPU Profile Analysis</h3>
                <p>The CPU profiles show the following hotspots:</p>
            """
            
            for profile in cpu_profiles:
                if not profile.get('top_functions'):
                    continue
                
                top_functions = pd.DataFrame(profile['top_functions'])
                top_3 = top_functions.nlargest(3, 'flat_pct')
                
                html_report += f"<p><strong>{profile['filename']}</strong>:</p><ul>"
                for _, row in top_3.iterrows():
                    html_report += f"<li>{row['function']}: {row['flat_pct']}% ({row['flat_val']})</li>"
                html_report += "</ul>"
            
            html_report += """
                <p><strong>Recommendations:</strong></p>
                <ul>
                    <li>Consider optimizing the most time-consuming functions identified above.</li>
                    <li>Look for opportunities to reduce allocations in hot code paths.</li>
                    <li>Consider using more efficient algorithms or data structures for critical operations.</li>
                </ul>
            """
        
        # Heap profile analysis
        if heap_profiles:
            html_report += """
                <h3>Memory Profile Analysis</h3>
                <p>The heap profiles show the following memory allocation hotspots:</p>
            """
            
            for profile in heap_profiles:
                if not profile.get('top_functions'):
                    continue
                
                top_functions = pd.DataFrame(profile['top_functions'])
                top_3 = top_functions.nlargest(3, 'flat_pct')
                
                html_report += f"<p><strong>{profile['filename']}</strong>:</p><ul>"
                for _, row in top_3.iterrows():
                    html_report += f"<li>{row['function']}: {row['flat_pct']}% ({row['flat_val']})</li>"
                html_report += "</ul>"
            
            html_report += """
                <p><strong>Recommendations:</strong></p>
                <ul>
                    <li>Review memory allocation patterns in the functions identified above.</li>
                    <li>Consider using object pools for frequently allocated objects.</li>
                    <li>Look for opportunities to reduce garbage collection pressure by minimizing allocations.</li>
                    <li>Consider using value types instead of pointers where appropriate.</li>
                </ul>
            """
        
        html_report += """
            </div>
        </body>
        </html>
        """
        
        html_report_file = os.path.join(self.report_dir, f'pprof_report_{datetime.now().strftime("%Y%m%d")}.html')
        with open(html_report_file, 'w') as f:
            f.write(html_report)
        
        return report_file

def main():
    """Main function to run the pprof analyzer."""
    parser = argparse.ArgumentParser(description='Analyze Go pprof profiles')
    parser.add_argument('--input', '-i', required=True, help='Directory containing pprof profile files')
    parser.add_argument('--output', '-o', required=True, help='Directory to save analysis results')
    
    args = parser.parse_args()
    
    analyzer = PprofAnalyzer(args.input, args.output)
    
    print("Analyzing pprof profiles...")
    results = analyzer.analyze_profiles()
    print(f"Analyzed {len(results['profiles'])} pprof profiles")
    
    print("Generating top functions plots...")
    plot_files = analyzer.generate_top_functions_plots()
    print(f"Generated {len(plot_files)} plot files")
    
    print("Generating summary report...")
    report_file = analyzer.generate_summary_report()
    print(f"Generated summary report: {report_file}")
    
    print("Analysis complete!")

if __name__ == "__main__":
    main() 