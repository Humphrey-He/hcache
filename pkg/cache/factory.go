package cache

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// New creates a new cache instance with the provided configuration.
// If config is nil, default configuration will be used.
//
// New 创建一个具有提供的配置的新缓存实例。
// 如果config为nil，将使用默认配置。
//
// Parameters:
//   - config: The configuration to use for the cache
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the cache creation fails
func New(config *Config) (ICache, error) {
	if config == nil {
		config = NewDefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cache configuration: %w", err)
	}

	// The actual implementation will be provided by an internal package
	// This is just a placeholder for the public API
	//
	// 实际实现将由内部包提供
	// 这只是公共API的占位符
	return nil, fmt.Errorf("not implemented yet")
}

// NewFromJSON creates a new cache from a JSON configuration file.
// The JSON document must represent a valid cache configuration.
//
// NewFromJSON 从JSON配置文件创建新的缓存。
// JSON文档必须表示有效的缓存配置。
//
// Parameters:
//   - reader: An io.Reader providing the JSON configuration
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the configuration is invalid or the cache creation fails
func NewFromJSON(reader io.Reader) (ICache, error) {
	config := NewDefaultConfig()
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode JSON configuration: %w", err)
	}

	return New(config)
}

// NewFromYAML creates a new cache from a YAML configuration file.
// The YAML document must represent a valid cache configuration.
//
// NewFromYAML 从YAML配置文件创建新的缓存。
// YAML文档必须表示有效的缓存配置。
//
// Parameters:
//   - reader: An io.Reader providing the YAML configuration
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the configuration is invalid or the cache creation fails
func NewFromYAML(reader io.Reader) (ICache, error) {
	config := NewDefaultConfig()
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode YAML configuration: %w", err)
	}

	return New(config)
}

// NewFromFile creates a new cache from a configuration file (JSON or YAML).
// The file format is determined by the file extension (.json, .yaml, or .yml).
//
// NewFromFile 从配置文件（JSON或YAML）创建新的缓存。
// 文件格式由文件扩展名确定（.json、.yaml或.yml）。
//
// Parameters:
//   - filename: The path to the configuration file
//
// Returns:
//   - ICache: The created cache instance
//   - error: An error if the file cannot be read, the format is unsupported,
//     the configuration is invalid, or the cache creation fails
func NewFromFile(filename string) (ICache, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open configuration file: %w", err)
	}
	defer file.Close()

	// Determine file type based on extension
	// 根据扩展名确定文件类型
	switch {
	case hasExtension(filename, ".json"):
		return NewFromJSON(file)
	case hasExtension(filename, ".yaml"), hasExtension(filename, ".yml"):
		return NewFromYAML(file)
	default:
		return nil, fmt.Errorf("unsupported configuration file format: %s", filename)
	}
}

// hasExtension checks if a filename has the specified extension.
// The comparison is case-sensitive.
//
// hasExtension 检查文件名是否具有指定的扩展名。
// 比较区分大小写。
//
// Parameters:
//   - filename: The filename to check
//   - ext: The extension to check for (including the dot)
//
// Returns:
//   - bool: True if the filename has the specified extension
func hasExtension(filename, ext string) bool {
	if len(filename) < len(ext) {
		return false
	}
	return filename[len(filename)-len(ext):] == ext
}
