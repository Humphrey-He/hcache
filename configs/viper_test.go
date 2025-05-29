// Package configs provides configuration structures and utilities for HCache.
// This file contains tests for the Viper-based configuration functionality.
//
// Package configs 提供HCache的配置结构和工具。
// 本文件包含基于Viper的配置功能的测试。
package configs

import (
	"strings"
	"testing"
	"time"
)

// TestViperConfigWithReader tests the Viper configuration loading using a reader
// instead of actual files to avoid filesystem dependencies. It verifies that
// configuration values are correctly parsed from YAML content.
//
// TestViperConfigWithReader 使用读取器而不是实际文件测试Viper配置加载，
// 以避免文件系统依赖。它验证配置值是否正确地从YAML内容解析。
func TestViperConfigWithReader(t *testing.T) {
	// Create a YAML config as a string
	// 创建一个YAML配置字符串
	yamlConfig := `
cache:
  enable: true
  name: "test-cache"
  max_entries: 1000
  max_memory_bytes: 536870912
  default_ttl: 60s
  cleanup_interval: 15s
storage:
  engine: "in-memory"
  shard_count: 64
  enable_ttl_tracking: true
eviction:
  policy: "lru"
  batch_size: 64
`

	// Load config from reader
	// 从读取器加载配置
	reader := strings.NewReader(yamlConfig)
	config, err := LoadFromReader(reader, "yaml")
	if err != nil {
		t.Fatalf("Failed to load config from reader: %v", err)
	}

	// Verify config values
	// 验证配置值
	if config.Cache.MaxEntries != 1000 {
		t.Errorf("Expected Cache.MaxEntries to be 1000, got %d", config.Cache.MaxEntries)
	}
	if config.Cache.Name != "test-cache" {
		t.Errorf("Expected Cache.Name to be 'test-cache', got '%s'", config.Cache.Name)
	}
	if config.Storage.ShardCount != 64 {
		t.Errorf("Expected Storage.ShardCount to be 64, got %d", config.Storage.ShardCount)
	}
	if config.Eviction.Policy != "lru" {
		t.Errorf("Expected Eviction.Policy to be 'lru', got '%s'", config.Eviction.Policy)
	}
	if config.Cache.DefaultTTL != 60*time.Second {
		t.Errorf("Expected Cache.DefaultTTL to be 60s, got %s", config.Cache.DefaultTTL)
	}
}

// TestConfigsEqual tests the configsEqual helper function to ensure it correctly
// identifies when two configurations are equal or different.
//
// TestConfigsEqual 测试configsEqual辅助函数，确保它能正确识别
// 两个配置何时相等或不同。
func TestConfigsEqual(t *testing.T) {
	config1 := DefaultConfig()
	config2 := DefaultConfig()

	// Same configs should be equal
	// 相同的配置应该相等
	if !configsEqual(config1, config2) {
		t.Error("configsEqual() returned false for identical configs")
	}

	// Different configs should not be equal
	// 不同的配置不应该相等
	config2.Cache.MaxEntries = 1000
	if configsEqual(config1, config2) {
		t.Error("configsEqual() returned true for different configs")
	}
}
