// Package configs provides configuration structures and utilities for HCache.
// This file contains tests for the configuration functionality.
//
// Package configs 提供HCache的配置结构和工具。
// 本文件包含配置功能的测试。
package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultConfig verifies that DefaultConfig returns a properly initialized Config
// with the expected default values for important settings.
//
// TestDefaultConfig 验证DefaultConfig返回一个正确初始化的Config，
// 包含重要设置的预期默认值。
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test default values
	// 测试默认值
	if config.Cache.MaxEntries != 500000 {
		t.Errorf("Expected Cache.MaxEntries to be 500000, got %d", config.Cache.MaxEntries)
	}
	if config.Storage.ShardCount != 256 {
		t.Errorf("Expected Storage.ShardCount to be 256, got %d", config.Storage.ShardCount)
	}
	if config.Eviction.Policy != "lfu" {
		t.Errorf("Expected Eviction.Policy to be 'lfu', got '%s'", config.Eviction.Policy)
	}
}

// TestLoadAndSaveConfig tests the ability to save and load configuration
// to and from files in both YAML and JSON formats.
//
// TestLoadAndSaveConfig 测试将配置保存到文件和从文件加载配置的能力，
// 包括YAML和JSON两种格式。
func TestLoadAndSaveConfig(t *testing.T) {
	// Create a temporary directory for test files
	// 创建测试文件的临时目录
	tempDir, err := os.MkdirTemp("", "hcache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test YAML
	// 测试YAML
	yamlPath := filepath.Join(tempDir, "config.yaml")
	config := DefaultConfig()
	config.Cache.MaxEntries = 1000
	config.Storage.ShardCount = 64
	config.Eviction.Policy = "lru"

	// Save config
	// 保存配置
	if err := config.SaveToFile(yamlPath); err != nil {
		t.Fatalf("Failed to save YAML config: %v", err)
	}

	// Load config
	// 加载配置
	loadedConfig, err := LoadFromFile(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load YAML config: %v", err)
	}

	// Verify loaded config
	// 验证加载的配置
	if loadedConfig.Cache.MaxEntries != 1000 {
		t.Errorf("Expected Cache.MaxEntries to be 1000, got %d", loadedConfig.Cache.MaxEntries)
	}
	if loadedConfig.Storage.ShardCount != 64 {
		t.Errorf("Expected Storage.ShardCount to be 64, got %d", loadedConfig.Storage.ShardCount)
	}
	if loadedConfig.Eviction.Policy != "lru" {
		t.Errorf("Expected Eviction.Policy to be 'lru', got '%s'", loadedConfig.Eviction.Policy)
	}

	// Test JSON
	// 测试JSON
	jsonPath := filepath.Join(tempDir, "config.json")
	config.Cache.MaxEntries = 2000
	config.Storage.ShardCount = 128
	config.Eviction.Policy = "fifo"

	// Save config
	// 保存配置
	if err := config.SaveToFile(jsonPath); err != nil {
		t.Fatalf("Failed to save JSON config: %v", err)
	}

	// Load config
	// 加载配置
	loadedConfig, err = LoadFromFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to load JSON config: %v", err)
	}

	// Verify loaded config
	// 验证加载的配置
	if loadedConfig.Cache.MaxEntries != 2000 {
		t.Errorf("Expected Cache.MaxEntries to be 2000, got %d", loadedConfig.Cache.MaxEntries)
	}
	if loadedConfig.Storage.ShardCount != 128 {
		t.Errorf("Expected Storage.ShardCount to be 128, got %d", loadedConfig.Storage.ShardCount)
	}
	if loadedConfig.Eviction.Policy != "fifo" {
		t.Errorf("Expected Eviction.Policy to be 'fifo', got '%s'", loadedConfig.Eviction.Policy)
	}
}

// TestValidate tests the Validate method to ensure it correctly identifies
// valid and invalid configurations according to the defined constraints.
//
// TestValidate 测试Validate方法，确保它能根据定义的约束
// 正确识别有效和无效的配置。
func TestValidate(t *testing.T) {
	tests := []struct {
		name        string        // Test case name / 测试用例名称
		modifyFunc  func(*Config) // Function to modify config / 修改配置的函数
		expectError bool          // Whether validation should fail / 验证是否应该失败
	}{
		{
			name:        "Valid default config",
			modifyFunc:  func(c *Config) {},
			expectError: false,
		},
		{
			name: "Invalid cache.max_entries",
			modifyFunc: func(c *Config) {
				c.Cache.MaxEntries = -1
			},
			expectError: true,
		},
		{
			name: "Invalid storage.shard_count not power of 2",
			modifyFunc: func(c *Config) {
				c.Storage.ShardCount = 100
			},
			expectError: true,
		},
		{
			name: "Invalid eviction.policy",
			modifyFunc: func(c *Config) {
				c.Eviction.Policy = "invalid"
			},
			expectError: true,
		},
		{
			name: "Invalid log.level",
			modifyFunc: func(c *Config) {
				c.Log.Level = "invalid"
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := DefaultConfig()
			test.modifyFunc(config)
			err := config.Validate()
			if test.expectError && err == nil {
				t.Error("Expected validation error, but got nil")
			}
			if !test.expectError && err != nil {
				t.Errorf("Expected no validation error, but got: %v", err)
			}
		})
	}
}

// TestIsPowerOfTwo tests the isPowerOfTwo helper function with various inputs
// to ensure it correctly identifies numbers that are powers of 2.
//
// TestIsPowerOfTwo 使用各种输入测试isPowerOfTwo辅助函数，
// 确保它能正确识别2的幂数。
func TestIsPowerOfTwo(t *testing.T) {
	testCases := []struct {
		n        int  // Input number / 输入数字
		expected bool // Expected result / 预期结果
	}{
		{0, false},
		{1, true},
		{2, true},
		{3, false},
		{4, true},
		{5, false},
		{8, true},
		{10, false},
		{16, true},
		{100, false},
		{128, true},
		{256, true},
		{1000, false},
		{1024, true},
	}

	for _, tc := range testCases {
		result := isPowerOfTwo(tc.n)
		if result != tc.expected {
			t.Errorf("isPowerOfTwo(%d) = %v, expected %v", tc.n, result, tc.expected)
		}
	}
}
