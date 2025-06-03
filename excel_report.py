#!/usr/bin/env python3
"""
简易Excel报告生成器

此脚本从测试结果CSV文件创建Excel报告。
"""

import os
import glob
import pandas as pd
import datetime
from pathlib import Path

# 设置目录
RESULTS_DIR = '../results/hitratio'
RUN1_DIR = '20250603_1'
RUN2_DIR = 'run2'
OUTPUT_FILE = '../results/hitratio/test_results.xlsx'

def load_data(run_dir, run_label):
    """从CSV文件加载数据"""
    data = {}
    csv_files = glob.glob(os.path.join(run_dir, '*.csv'))
    
    for file_path in csv_files:
        pattern_name = Path(file_path).stem
        if pattern_name != 'summary':
            try:
                df = pd.read_csv(file_path)
                
                # 添加运行标识
                df['Run'] = run_label
                
                # 转换命中率为浮点数
                if 'HitRatio' in df.columns:
                    df['HitRatio'] = df['HitRatio'].astype(float)
                
                data[pattern_name] = df
            except Exception as e:
                print(f"Error loading {file_path}: {e}")
    
    return data

def generate_excel_report():
    """生成Excel报告"""
    # 加载数据
    run1_data = load_data(os.path.join(RESULTS_DIR, RUN1_DIR), "Run 1")
    run2_data = load_data(os.path.join(RESULTS_DIR, RUN2_DIR), "Run 2")
    
    # 获取测试模式
    test_patterns = list(set(list(run1_data.keys()) + list(run2_data.keys())))
    
    # 创建Excel写入器
    writer = pd.ExcelWriter(OUTPUT_FILE, engine='xlsxwriter')
    
    # 创建摘要数据框
    summary_data = []
    
    for pattern in sorted(test_patterns):
        # 跳过任一运行中缺失的模式
        if pattern not in run1_data or pattern not in run2_data:
            continue
        
        run1_df = run1_data[pattern]
        run2_df = run2_data[pattern]
        
        # 合并两次运行的数据
        for _, r1 in run1_df.iterrows():
            for _, r2 in run2_df.iterrows():
                if r1['Policy'] == r2['Policy'] and r1['CacheSize'] == r2['CacheSize']:
                    summary_data.append({
                        'Test Pattern': pattern,
                        'Policy': r1['Policy'],
                        'Cache Size': r1['CacheSize'],
                        'Run 1 Hit Ratio': r1['HitRatio'],
                        'Run 2 Hit Ratio': r2['HitRatio'],
                        'Difference': r2['HitRatio'] - r1['HitRatio']
                    })
    
    # 创建摘要数据框
    summary_df = pd.DataFrame(summary_data)
    
    # 写入摘要表
    summary_df.to_excel(writer, sheet_name='Summary', index=False)
    
    # 为每个测试模式创建单独的表
    for pattern in sorted(test_patterns):
        # 跳过任一运行中缺失的模式
        if pattern not in run1_data or pattern not in run2_data:
            continue
        
        # 合并数据
        pattern_data = []
        run1_df = run1_data[pattern]
        run2_df = run2_data[pattern]
        
        for _, r1 in run1_df.iterrows():
            for _, r2 in run2_df.iterrows():
                if r1['Policy'] == r2['Policy'] and r1['CacheSize'] == r2['CacheSize']:
                    pattern_data.append({
                        'Policy': r1['Policy'],
                        'Cache Size': r1['CacheSize'],
                        'Run 1 Hit Ratio': r1['HitRatio'],
                        'Run 2 Hit Ratio': r2['HitRatio']
                    })
        
        # 创建并写入模式数据框
        pattern_df = pd.DataFrame(pattern_data)
        pattern_df.to_excel(writer, sheet_name=pattern, index=False)
    
    # 关闭写入器
    writer.close()
    
    print(f"Excel报告已生成: {OUTPUT_FILE}")

if __name__ == "__main__":
    # 确保输出目录存在
    os.makedirs(os.path.dirname(OUTPUT_FILE), exist_ok=True)
    
    # 生成报告
    generate_excel_report() 