#!/usr/bin/env python3
"""
Custom Hit Ratio Visualization Script

This script visualizes hit ratio test results from two different test runs
and creates comparison visualizations.

自定义命中率可视化脚本
此脚本可视化两次不同测试运行的命中率测试结果并创建比较可视化。
"""

import os
import glob
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import numpy as np
from pathlib import Path
import datetime

# Set plot style
# 设置绘图样式
sns.set(style="whitegrid")
plt.rcParams.update({'font.size': 12})

class CustomHitRatioVisualizer:
    """
    A class to visualize and compare hit ratio test results from two runs.
    
    用于可视化和比较两次运行的命中率测试结果的类。
    """
    
    def __init__(self, 
                 results_dir='tests/results/hitratio', 
                 run1_dir='20250603_1',
                 run2_dir='run2',
                 output_dir='tests/results/hitratio/visualizations'):
        """
        Initialize the visualizer with directories for input and output.
        
        Parameters:
        - results_dir: Base directory containing results
        - run1_dir: Directory for first test run
        - run2_dir: Directory for second test run
        - output_dir: Directory to save visualization outputs
        
        使用输入和输出目录初始化可视化器。
        
        参数:
        - results_dir: 包含结果的基本目录
        - run1_dir: 第一次测试运行的目录
        - run2_dir: 第二次测试运行的目录
        - output_dir: 保存可视化输出的目录
        """
        self.results_dir = results_dir
        self.run1_dir = os.path.join(results_dir, run1_dir)
        self.run2_dir = os.path.join(results_dir, run2_dir)
        self.output_dir = output_dir
        
        # Create output directory if it doesn't exist
        # 如果输出目录不存在，则创建它
        os.makedirs(output_dir, exist_ok=True)
        
        # Load data
        # 加载数据
        self.run1_data = self._load_data(self.run1_dir, "Run 1")
        self.run2_data = self._load_data(self.run2_dir, "Run 2")
        
        # Get test patterns
        # 获取测试模式
        self.test_patterns = list(set(list(self.run1_data.keys()) + list(self.run2_data.keys())))
        
    def _load_data(self, run_dir, run_label):
        """
        Load data from CSV files in the run directory.
        
        Returns:
        - Dictionary mapping test pattern names to pandas DataFrames
        
        从运行目录中的CSV文件加载数据。
        
        返回:
        - 将测试模式名称映射到pandas DataFrame的字典
        """
        data = {}
        csv_files = glob.glob(os.path.join(run_dir, '*.csv'))
        
        for file_path in csv_files:
            pattern_name = Path(file_path).stem
            if pattern_name != 'summary':
                try:
                    df = pd.read_csv(file_path)
                    
                    # Add a column to identify the run
                    # 添加一列以标识运行
                    df['Run'] = run_label
                    
                    # Convert hit ratio to float if it's not already
                    # 如果命中率不是浮点数，则将其转换为浮点数
                    if 'HitRatio' in df.columns:
                        df['HitRatio'] = df['HitRatio'].astype(float)
                    
                    data[pattern_name] = df
                except Exception as e:
                    print(f"Error loading {file_path}: {e}")
        
        return data
    
    def create_comparison_bar_charts(self):
        """
        Create comparison bar charts for each test pattern showing hit ratios by policy,
        cache size, and run.
        
        为每个测试模式创建比较条形图，显示按策略、缓存大小和运行的命中率。
        """
        for pattern in self.test_patterns:
            # Skip if pattern is missing from either run
            # 如果模式在任一运行中缺失，则跳过
            if pattern not in self.run1_data or pattern not in self.run2_data:
                continue
                
            plt.figure(figsize=(16, 10))
            
            # Combine data from both runs
            # 合并两次运行的数据
            combined_df = pd.concat([self.run1_data[pattern], self.run2_data[pattern]])
            
            # Create grouped bar chart
            # 创建分组条形图
            chart = sns.catplot(
                x='Policy', 
                y='HitRatio', 
                hue='Run',
                col='CacheSize',
                data=combined_df,
                kind='bar',
                height=8,
                aspect=0.8,
                palette='viridis'
            )
            
            chart.fig.suptitle(f'Hit Ratio Comparison by Policy and Cache Size - {pattern}', fontsize=16)
            chart.set_axis_labels('Eviction Policy', 'Hit Ratio (%)')
            chart.fig.subplots_adjust(top=0.9)
            
            # Save figure
            # 保存图形
            output_path = os.path.join(self.output_dir, f'{pattern}_comparison_chart.png')
            plt.savefig(output_path, dpi=300)
            plt.close()
            
            print(f"Created comparison chart for {pattern} at {output_path}")
    
    def create_policy_comparison(self):
        """
        Create a comparison chart of different policies across all test patterns.
        
        创建所有测试模式中不同策略的比较图。
        """
        # Prepare data for comparison
        # 准备比较数据
        comparison_data = []
        
        # Process run1 data
        # 处理第一次运行数据
        for pattern, df in self.run1_data.items():
            for _, row in df.iterrows():
                comparison_data.append({
                    'Pattern': pattern,
                    'Policy': row['Policy'],
                    'CacheSize': row['CacheSize'],
                    'HitRatio': row['HitRatio'],
                    'Run': 'Run 1'
                })
        
        # Process run2 data
        # 处理第二次运行数据
        for pattern, df in self.run2_data.items():
            for _, row in df.iterrows():
                comparison_data.append({
                    'Pattern': pattern,
                    'Policy': row['Policy'],
                    'CacheSize': row['CacheSize'],
                    'HitRatio': row['HitRatio'],
                    'Run': 'Run 2'
                })
        
        comparison_df = pd.DataFrame(comparison_data)
        
        if comparison_df.empty:
            print("No comparison data available")
            return
        
        # Create comparison charts for each cache size
        # 为每个缓存大小创建比较图
        cache_sizes = comparison_df['CacheSize'].unique()
        
        for size in cache_sizes:
            plt.figure(figsize=(18, 12))
            
            size_data = comparison_df[comparison_df['CacheSize'] == size]
            
            chart = sns.catplot(
                x='Pattern', 
                y='HitRatio', 
                hue='Policy',
                col='Run',
                data=size_data,
                kind='bar',
                height=8,
                aspect=1.2,
                palette='Set2'
            )
            
            chart.fig.suptitle(f'Policy Comparison Across Test Patterns - Cache Size: {size}', fontsize=16)
            chart.set_axis_labels('Test Pattern', 'Hit Ratio (%)')
            chart.set_xticklabels(rotation=45)
            chart.fig.subplots_adjust(top=0.9)
            
            # Save figure
            # 保存图形
            output_path = os.path.join(self.output_dir, f'policy_comparison_size_{size}.png')
            plt.savefig(output_path, dpi=300)
            plt.close()
            
            print(f"Created policy comparison chart for cache size {size} at {output_path}")
    
    def create_heatmap(self):
        """
        Create a heatmap showing hit ratios across all test patterns and policies.
        
        创建一个热图，显示所有测试模式和策略的命中率。
        """
        # Prepare data for heatmap
        # 准备热图数据
        for size in sorted(set(df['CacheSize'].unique()[0] for df in self.run1_data.values() if len(df) > 0)):
            for run_label, run_data in [("Run 1", self.run1_data), ("Run 2", self.run2_data)]:
                # Extract data for this cache size
                # 提取此缓存大小的数据
                heatmap_data = {}
                
                for pattern, df in run_data.items():
                    if len(df) == 0:
                        continue
                    size_df = df[df['CacheSize'] == size]
                    if len(size_df) > 0:
                        heatmap_data[pattern] = dict(zip(size_df['Policy'], size_df['HitRatio']))
                
                if not heatmap_data:
                    continue
                
                # Create DataFrame for heatmap
                # 为热图创建DataFrame
                patterns = list(heatmap_data.keys())
                policies = ['lru', 'lfu', 'fifo', 'random']
                
                heatmap_values = []
                for policy in policies:
                    policy_values = []
                    for pattern in patterns:
                        policy_values.append(heatmap_data[pattern].get(policy, 0))
                    heatmap_values.append(policy_values)
                
                heatmap_df = pd.DataFrame(heatmap_values, index=policies, columns=patterns)
                
                # Create heatmap
                # 创建热图
                plt.figure(figsize=(12, 8))
                
                ax = sns.heatmap(
                    heatmap_df,
                    annot=True,
                    fmt=".2f",
                    cmap="YlGnBu",
                    cbar_kws={'label': 'Hit Ratio (%)'}
                )
                
                plt.title(f'Hit Ratio Heatmap - Cache Size: {size} - {run_label}', fontsize=16)
                plt.xlabel('Test Pattern', fontsize=14)
                plt.ylabel('Eviction Policy', fontsize=14)
                plt.tight_layout()
                
                # Save figure
                # 保存图形
                output_path = os.path.join(self.output_dir, f'heatmap_size_{size}_{run_label.replace(" ", "_")}.png')
                plt.savefig(output_path, dpi=300)
                plt.close()
                
                print(f"Created heatmap for cache size {size} - {run_label} at {output_path}")
    
    def generate_comparison_report(self):
        """
        Generate a Markdown report comparing the results of the two runs.
        
        生成比较两次运行结果的Markdown报告。
        """
        report_path = os.path.join(self.output_dir, 'comparison_report.md')
        
        with open(report_path, 'w') as f:
            f.write("# HCache 命中率测试比较报告\n\n")
            f.write(f"生成时间: {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n")
            
            f.write("## 测试概述\n\n")
            f.write("本报告比较了HCache的两次命中率测试结果，包括不同策略和缓存大小的性能。\n\n")
            
            f.write("## 测试模式与策略\n\n")
            f.write("### 测试模式\n\n")
            f.write("- **竞争抵抗 (Contention Resistance)**: 测试当多种访问模式竞争缓存空间时的性能\n")
            f.write("- **搜索模式 (Search Pattern)**: 模拟搜索引擎查询模式，包含少量热门项目和大量罕见项目\n")
            f.write("- **数据库模式 (Database Pattern)**: 模拟数据库访问模式，包括记录访问和索引查找\n")
            f.write("- **循环模式 (Looping Pattern)**: 模拟以循环模式重复访问相同数据集\n")
            f.write("- **CODASYL模式**: 模拟网络数据库模式，数据在图结构中被访问\n\n")
            
            f.write("### 缓存策略\n\n")
            f.write("- **LRU (最近最少使用)**: 首先淘汰最近最少访问的项目\n")
            f.write("- **LFU (最不经常使用)**: 首先淘汰访问频率最低的项目\n")
            f.write("- **FIFO (先进先出)**: 首先淘汰最早的项目，不考虑访问频率\n")
            f.write("- **Random (随机)**: 随机选择要淘汰的项目，作为基准线\n\n")
            
            f.write("## 测试结果比较\n\n")
            
            # Add links to the generated visualizations
            # 添加到生成的可视化的链接
            f.write("### 可视化结果\n\n")
            f.write("#### 策略比较图\n\n")
            
            for pattern in self.test_patterns:
                f.write(f"- [{pattern} 策略比较](../{pattern}_comparison_chart.png)\n")
            
            f.write("\n#### 缓存大小比较图\n\n")
            cache_sizes = set()
            for run_data in [self.run1_data, self.run2_data]:
                for df in run_data.values():
                    if len(df) > 0:
                        cache_sizes.update(df['CacheSize'].unique())
            
            for size in sorted(cache_sizes):
                f.write(f"- [缓存大小 {size} 比较](../policy_comparison_size_{size}.png)\n")
            
            f.write("\n#### 热图\n\n")
            for size in sorted(cache_sizes):
                f.write(f"- [缓存大小 {size} - 运行1](../heatmap_size_{size}_Run_1.png)\n")
                f.write(f"- [缓存大小 {size} - 运行2](../heatmap_size_{size}_Run_2.png)\n")
            
            f.write("\n## 结论\n\n")
            f.write("通过比较两次测试结果，我们可以得出以下结论：\n\n")
            f.write("1. 测试结果的一致性：两次测试的结果是否一致，表明测试的可重复性\n")
            f.write("2. 不同策略的表现：在不同测试模式下，各种策略的性能比较\n")
            f.write("3. 缓存大小的影响：增加缓存大小对命中率的影响\n")
            f.write("4. 推荐配置：基于测试结果，推荐最佳的缓存策略和大小配置\n\n")
            
            f.write("### 详细分析\n\n")
            f.write("请根据生成的图表进行详细分析...\n")
        
        print(f"Generated comparison report at {report_path}")
    
    def create_all_visualizations(self):
        """
        Create all visualizations and generate the comparison report.
        
        创建所有可视化并生成比较报告。
        """
        print("Creating comparison bar charts...")
        self.create_comparison_bar_charts()
        
        print("\nCreating policy comparison charts...")
        self.create_policy_comparison()
        
        print("\nCreating heatmaps...")
        self.create_heatmap()
        
        print("\nGenerating comparison report...")
        self.generate_comparison_report()
        
        print("\nAll visualizations and report generated successfully!")


if __name__ == "__main__":
    # Set the directories
    # 设置目录
    results_dir = 'tests/results/hitratio'
    run1_dir = '20250603_1'
    run2_dir = 'run2'
    output_dir = 'tests/results/hitratio/visualizations'
    
    # Create the visualizer and generate all visualizations
    # 创建可视化器并生成所有可视化
    visualizer = CustomHitRatioVisualizer(
        results_dir=results_dir,
        run1_dir=run1_dir,
        run2_dir=run2_dir,
        output_dir=output_dir
    )
    
    visualizer.create_all_visualizations() 