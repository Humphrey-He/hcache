#!/usr/bin/env python3
"""
Hit Ratio Visualization Script

This script visualizes hit ratio test results from CSV files.
It creates bar charts and comparison plots for different cache policies and access patterns.

使用此脚本可视化来自CSV文件的命中率测试结果。
它为不同的缓存策略和访问模式创建条形图和比较图。
"""

import os
import glob
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import numpy as np
from pathlib import Path

# Set plot style
# 设置绘图样式
sns.set(style="whitegrid")
plt.rcParams.update({'font.size': 12})

class HitRatioVisualizer:
    """
    A class to visualize hit ratio test results.
    
    用于可视化命中率测试结果的类。
    """
    
    def __init__(self, results_dir='results/hitratio', output_dir='results/hitratio/visualizations'):
        """
        Initialize the visualizer with directories for input and output.
        
        Parameters:
        - results_dir: Directory containing CSV result files
        - output_dir: Directory to save visualization outputs
        
        使用输入和输出目录初始化可视化器。
        
        参数:
        - results_dir: 包含CSV结果文件的目录
        - output_dir: 保存可视化输出的目录
        """
        self.results_dir = results_dir
        self.output_dir = output_dir
        
        # Create output directory if it doesn't exist
        # 如果输出目录不存在，则创建它
        os.makedirs(output_dir, exist_ok=True)
        
        # Load data
        # 加载数据
        self.data = self._load_data()
        
    def _load_data(self):
        """
        Load data from CSV files in the results directory.
        
        Returns:
        - Dictionary mapping test pattern names to pandas DataFrames
        
        从结果目录中的CSV文件加载数据。
        
        返回:
        - 将测试模式名称映射到pandas DataFrame的字典
        """
        data = {}
        csv_files = glob.glob(os.path.join(self.results_dir, '*.csv'))
        
        for file_path in csv_files:
            pattern_name = Path(file_path).stem
            if pattern_name != 'summary':
                try:
                    df = pd.read_csv(file_path)
                    # Convert hit ratio to float if it's not already
                    # 如果命中率不是浮点数，则将其转换为浮点数
                    df['HitRatio'] = df['HitRatio'].astype(float)
                    data[pattern_name] = df
                except Exception as e:
                    print(f"Error loading {file_path}: {e}")
        
        return data
    
    def create_bar_charts(self):
        """
        Create bar charts for each test pattern showing hit ratios by policy and cache size.
        
        为每个测试模式创建条形图，显示按策略和缓存大小的命中率。
        """
        for pattern, df in self.data.items():
            plt.figure(figsize=(12, 8))
            
            # Create grouped bar chart
            # 创建分组条形图
            chart = sns.barplot(
                x='Policy', 
                y='HitRatio', 
                hue='CacheSize', 
                data=df,
                palette='viridis'
            )
            
            plt.title(f'Hit Ratio by Policy and Cache Size - {pattern}', fontsize=16)
            plt.xlabel('Eviction Policy', fontsize=14)
            plt.ylabel('Hit Ratio (%)', fontsize=14)
            plt.legend(title='Cache Size (entries)')
            plt.grid(True, linestyle='--', alpha=0.7)
            
            # Add value labels on top of bars
            # 在条形顶部添加值标签
            for p in chart.patches:
                chart.annotate(
                    f'{p.get_height():.2f}%',
                    (p.get_x() + p.get_width() / 2., p.get_height()),
                    ha='center', va='bottom',
                    fontsize=10
                )
            
            # Save figure
            # 保存图形
            output_path = os.path.join(self.output_dir, f'{pattern}_bar_chart.png')
            plt.tight_layout()
            plt.savefig(output_path, dpi=300)
            plt.close()
            
            print(f"Created bar chart for {pattern} at {output_path}")
    
    def create_policy_comparison(self):
        """
        Create a comparison chart of different policies across all test patterns.
        
        创建所有测试模式中不同策略的比较图。
        """
        # Prepare data for comparison
        # 准备比较数据
        comparison_data = []
        
        for pattern, df in self.data.items():
            for _, row in df.iterrows():
                comparison_data.append({
                    'Pattern': pattern,
                    'Policy': row['Policy'],
                    'CacheSize': row['CacheSize'],
                    'HitRatio': row['HitRatio']
                })
        
        comparison_df = pd.DataFrame(comparison_data)
        
        # Create comparison charts for each cache size
        # 为每个缓存大小创建比较图
        cache_sizes = comparison_df['CacheSize'].unique()
        
        for size in cache_sizes:
            plt.figure(figsize=(14, 10))
            
            size_data = comparison_df[comparison_df['CacheSize'] == size]
            
            chart = sns.barplot(
                x='Pattern', 
                y='HitRatio', 
                hue='Policy', 
                data=size_data,
                palette='Set2'
            )
            
            plt.title(f'Policy Comparison Across Test Patterns - Cache Size: {size}', fontsize=16)
            plt.xlabel('Test Pattern', fontsize=14)
            plt.ylabel('Hit Ratio (%)', fontsize=14)
            plt.legend(title='Eviction Policy')
            plt.grid(True, linestyle='--', alpha=0.7)
            plt.xticks(rotation=45)
            
            # Save figure
            # 保存图形
            output_path = os.path.join(self.output_dir, f'policy_comparison_size_{size}.png')
            plt.tight_layout()
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
        for size in self.data[list(self.data.keys())[0]]['CacheSize'].unique():
            # Extract data for this cache size
            # 提取此缓存大小的数据
            heatmap_data = {}
            
            for pattern, df in self.data.items():
                size_df = df[df['CacheSize'] == size]
                heatmap_data[pattern] = dict(zip(size_df['Policy'], size_df['HitRatio']))
            
            # Convert to DataFrame for heatmap
            # 转换为DataFrame用于热图
            heatmap_df = pd.DataFrame(heatmap_data).T
            
            # Create heatmap
            # 创建热图
            plt.figure(figsize=(12, 8))
            sns.heatmap(
                heatmap_df,
                annot=True,
                fmt='.2f',
                cmap='YlGnBu',
                linewidths=.5,
                cbar_kws={'label': 'Hit Ratio (%)'}
            )
            
            plt.title(f'Hit Ratio Heatmap - Cache Size: {size}', fontsize=16)
            plt.ylabel('Test Pattern', fontsize=14)
            plt.xlabel('Eviction Policy', fontsize=14)
            
            # Save figure
            # 保存图形
            output_path = os.path.join(self.output_dir, f'heatmap_size_{size}.png')
            plt.tight_layout()
            plt.savefig(output_path, dpi=300)
            plt.close()
            
            print(f"Created heatmap for cache size {size} at {output_path}")
    
    def create_radar_chart(self):
        """
        Create radar charts comparing policies across test patterns.
        
        创建雷达图，比较各测试模式中的策略。
        """
        # Get unique policies and patterns
        # 获取唯一的策略和模式
        policies = []
        patterns = []
        
        for pattern, df in self.data.items():
            patterns.append(pattern)
            for policy in df['Policy'].unique():
                if policy not in policies:
                    policies.append(policy)
        
        # Create radar charts for each cache size
        # 为每个缓存大小创建雷达图
        cache_sizes = self.data[list(self.data.keys())[0]]['CacheSize'].unique()
        
        for size in cache_sizes:
            fig, ax = plt.subplots(figsize=(10, 10), subplot_kw=dict(polar=True))
            
            # Number of variables
            # 变量数量
            N = len(patterns)
            
            # Compute angle for each axis
            # 计算每个轴的角度
            angles = np.linspace(0, 2 * np.pi, N, endpoint=False).tolist()
            angles += angles[:1]  # Close the polygon
            
            # Plot for each policy
            # 为每个策略绘图
            for policy in policies:
                values = []
                
                for pattern in patterns:
                    df = self.data[pattern]
                    policy_size_df = df[(df['Policy'] == policy) & (df['CacheSize'] == size)]
                    if not policy_size_df.empty:
                        values.append(policy_size_df['HitRatio'].values[0])
                    else:
                        values.append(0)
                
                # Close the polygon
                # 闭合多边形
                values += values[:1]
                
                # Plot values
                # 绘制值
                ax.plot(angles, values, linewidth=2, label=policy)
                ax.fill(angles, values, alpha=0.25)
            
            # Set labels
            # 设置标签
            ax.set_xticks(angles[:-1])
            ax.set_xticklabels(patterns)
            
            # Add legend and title
            # 添加图例和标题
            plt.legend(loc='upper right', bbox_to_anchor=(0.1, 0.1))
            plt.title(f'Policy Comparison Radar Chart - Cache Size: {size}', size=15)
            
            # Save figure
            # 保存图形
            output_path = os.path.join(self.output_dir, f'radar_chart_size_{size}.png')
            plt.tight_layout()
            plt.savefig(output_path, dpi=300)
            plt.close()
            
            print(f"Created radar chart for cache size {size} at {output_path}")
    
    def create_all_visualizations(self):
        """
        Create all visualizations.
        
        创建所有可视化。
        """
        if not self.data:
            print("No data available for visualization.")
            return
            
        self.create_bar_charts()
        self.create_policy_comparison()
        self.create_heatmap()
        self.create_radar_chart()
        
        print("All visualizations created successfully!")


if __name__ == "__main__":
    visualizer = HitRatioVisualizer()
    visualizer.create_all_visualizations() 