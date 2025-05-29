// Package configs provides configuration structures and utilities for HCache.
// This file implements Viper-based configuration management with hot reloading support.
//
// Package configs 提供HCache的配置结构和工具。
// 本文件实现基于Viper的配置管理，支持热重载。
package configs

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ViperConfig wraps a Config with Viper functionality for hot reloading.
// It provides thread-safe access to configuration and supports dynamic
// updates when the underlying configuration file changes.
//
// ViperConfig 使用Viper功能包装Config以支持热重载。
// 它提供对配置的线程安全访问，并支持在底层配置文件更改时进行动态更新。
type ViperConfig struct {
	*Config                     // Embedded configuration / 嵌入的配置
	viper       *viper.Viper    // Viper instance for configuration management / 用于配置管理的Viper实例
	configFile  string          // Path to the configuration file / 配置文件路径
	mu          sync.RWMutex    // Mutex for thread-safe access / 用于线程安全访问的互斥锁
	subscribers []func(*Config) // List of subscribers to notify on config changes / 配置更改时要通知的订阅者列表
}

// NewViperConfig creates a new ViperConfig.
// It loads configuration from the specified file and validates it.
//
// NewViperConfig 创建一个新的ViperConfig。
// 它从指定的文件加载配置并验证它。
//
// Parameters:
//   - configFile: Path to the configuration file
//
// Returns:
//   - *ViperConfig: A new ViperConfig instance
//   - error: An error if loading or validation fails
//
// 参数：
//   - configFile: 配置文件的路径
//
// 返回：
//   - *ViperConfig: 一个新的ViperConfig实例
//   - error: 如果加载或验证失败则返回错误
func NewViperConfig(configFile string) (*ViperConfig, error) {
	v := viper.New()

	// Set up viper
	// 设置viper
	v.SetConfigFile(configFile)
	ext := filepath.Ext(configFile)
	v.SetConfigType(strings.TrimPrefix(ext, "."))

	// Read the config file
	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Create config
	// 创建配置
	config := DefaultConfig()

	// Unmarshal the config file into the config struct
	// 将配置文件解析到配置结构中
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &ViperConfig{
		Config:      config,
		viper:       v,
		configFile:  configFile,
		subscribers: make([]func(*Config), 0),
	}, nil
}

// EnableHotReload enables hot reloading of the configuration file.
// When the configuration file changes, the configuration is automatically
// reloaded and all subscribers are notified.
//
// EnableHotReload 启用配置文件的热重载。
// 当配置文件更改时，配置会自动重新加载，并通知所有订阅者。
func (vc *ViperConfig) EnableHotReload() {
	vc.viper.WatchConfig()
	vc.viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Config file changed: %s", e.Name)

		// Create a new config
		// 创建新配置
		newConfig := DefaultConfig()
		if err := vc.viper.Unmarshal(newConfig); err != nil {
			log.Printf("Failed to unmarshal config: %v", err)
			return
		}

		// Validate the new config
		// 验证新配置
		if err := newConfig.Validate(); err != nil {
			log.Printf("Invalid configuration: %v", err)
			return
		}

		// Update the config
		// 更新配置
		vc.mu.Lock()
		vc.Config = newConfig
		subscribers := make([]func(*Config), len(vc.subscribers))
		copy(subscribers, vc.subscribers)
		vc.mu.Unlock()

		// Notify subscribers
		// 通知订阅者
		for _, subscriber := range subscribers {
			subscriber(newConfig)
		}
	})
}

// Subscribe adds a subscriber that will be notified when the configuration changes.
// The subscriber function is called with the new configuration as its argument.
//
// Subscribe 添加一个在配置更改时将被通知的订阅者。
// 订阅者函数将以新配置作为其参数被调用。
//
// Parameters:
//   - subscriber: A function to call when the configuration changes
//
// 参数：
//   - subscriber: 配置更改时要调用的函数
func (vc *ViperConfig) Subscribe(subscriber func(*Config)) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.subscribers = append(vc.subscribers, subscriber)
}

// Get returns the current configuration.
// This method is thread-safe and can be called concurrently.
//
// Get 返回当前配置。
// 此方法是线程安全的，可以并发调用。
//
// Returns:
//   - *Config: The current configuration
//
// 返回：
//   - *Config: 当前配置
func (vc *ViperConfig) Get() *Config {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.Config
}

// LoadViperConfig loads a configuration from a file using Viper.
// It optionally enables hot reloading based on the enableHotReload parameter.
//
// LoadViperConfig 使用Viper从文件加载配置。
// 它根据enableHotReload参数可选地启用热重载。
//
// Parameters:
//   - configFile: Path to the configuration file
//   - enableHotReload: Whether to enable hot reloading
//
// Returns:
//   - *ViperConfig: A new ViperConfig instance
//   - error: An error if loading fails
//
// 参数：
//   - configFile: 配置文件的路径
//   - enableHotReload: 是否启用热重载
//
// 返回：
//   - *ViperConfig: 一个新的ViperConfig实例
//   - error: 如果加载失败则返回错误
func LoadViperConfig(configFile string, enableHotReload bool) (*ViperConfig, error) {
	vc, err := NewViperConfig(configFile)
	if err != nil {
		return nil, err
	}

	if enableHotReload {
		vc.EnableHotReload()
	}

	return vc, nil
}

// LoadViperConfigWithWatcher loads a configuration from a file using Viper and sets up a watcher
// that periodically checks for changes in the configuration file.
// This is an alternative to fsnotify-based hot reloading and may be more reliable
// in environments where file system notifications are unreliable.
//
// LoadViperConfigWithWatcher 使用Viper从文件加载配置，并设置一个定期检查
// 配置文件变化的监视器。这是基于fsnotify的热重载的替代方案，在文件系统
// 通知不可靠的环境中可能更可靠。
//
// Parameters:
//   - configFile: Path to the configuration file
//   - watchInterval: How often to check for changes
//
// Returns:
//   - *ViperConfig: A new ViperConfig instance
//   - error: An error if loading fails
//
// 参数：
//   - configFile: 配置文件的路径
//   - watchInterval: 检查更改的频率
//
// 返回：
//   - *ViperConfig: 一个新的ViperConfig实例
//   - error: 如果加载失败则返回错误
func LoadViperConfigWithWatcher(configFile string, watchInterval time.Duration) (*ViperConfig, error) {
	vc, err := NewViperConfig(configFile)
	if err != nil {
		return nil, err
	}

	// Start a goroutine to watch for changes
	// 启动一个goroutine来监视更改
	go func() {
		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()

		for range ticker.C {
			// Check if the file has been modified
			// 检查文件是否已修改
			if err := vc.viper.ReadInConfig(); err != nil {
				log.Printf("Failed to read config file: %v", err)
				continue
			}

			// Create a new config
			// 创建新配置
			newConfig := DefaultConfig()
			if err := vc.viper.Unmarshal(newConfig); err != nil {
				log.Printf("Failed to unmarshal config: %v", err)
				continue
			}

			// Validate the new config
			// 验证新配置
			if err := newConfig.Validate(); err != nil {
				log.Printf("Invalid configuration: %v", err)
				continue
			}

			// Check if the config has changed
			// 检查配置是否已更改
			vc.mu.RLock()
			changed := !configsEqual(vc.Config, newConfig)
			vc.mu.RUnlock()

			if changed {
				log.Printf("Config file changed: %s", configFile)

				// Update the config
				// 更新配置
				vc.mu.Lock()
				vc.Config = newConfig
				subscribers := make([]func(*Config), len(vc.subscribers))
				copy(subscribers, vc.subscribers)
				vc.mu.Unlock()

				// Notify subscribers
				// 通知订阅者
				for _, subscriber := range subscribers {
					subscriber(newConfig)
				}
			}
		}
	}()

	return vc, nil
}

// configsEqual checks if two configs are equal.
// This is a simple implementation that just compares the string representation of the configs.
//
// configsEqual 检查两个配置是否相等。
// 这是一个简单的实现，只比较配置的字符串表示。
//
// Parameters:
//   - c1: First configuration to compare
//   - c2: Second configuration to compare
//
// Returns:
//   - bool: True if the configurations are equal, false otherwise
//
// 参数：
//   - c1: 要比较的第一个配置
//   - c2: 要比较的第二个配置
//
// 返回：
//   - bool: 如果配置相等则为true，否则为false
func configsEqual(c1, c2 *Config) bool {
	// This is a simple implementation that just compares the string representation of the configs.
	// In a real implementation, you might want to do a more sophisticated comparison.
	//
	// 这是一个简单的实现，只比较配置的字符串表示。
	// 在实际实现中，您可能希望进行更复杂的比较。
	return fmt.Sprintf("%v", c1) == fmt.Sprintf("%v", c2)
}
