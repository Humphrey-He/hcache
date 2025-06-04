package core

import (
	"fmt"
)

// Factory defines the interface for creating cache instances.
// This allows for different cache implementations to be created
// through a common interface.
type Factory interface {
	// Create creates a new cache instance with the given name and options.
	//
	// Parameters:
	//   - name: A unique identifier for the cache
	//   - options: Configuration options for the cache
	//
	// Returns:
	//   - Cache: A new cache instance
	//   - error: An error if creation fails
	Create(name string, options ...Option) (Cache, error)
}

// DefaultConfig returns a Config with reasonable default values.
// This can be used as a starting point for customization.
func DefaultConfig() *Config {
	return &Config{
		MaxEntryCount:   10000,
		MaxMemoryBytes:  0, // No limit by default
		DefaultTTL:      0, // No expiration by default
		EvictionPolicy:  "lru",
		Shards:          16,
		MetricsEnabled:  true,
		CleanupInterval: 0, // No automatic cleanup by default
	}
}

// ApplyOptions applies the given options to a config.
// This is a helper function for implementing the functional options pattern.
func ApplyOptions(config *Config, options ...Option) {
	for _, option := range options {
		option(config)
	}
}

// ValidateConfig checks if a configuration is valid.
// It returns an error if any configuration parameter is invalid.
func ValidateConfig(config *Config) error {
	if config.MaxEntryCount <= 0 && config.MaxMemoryBytes <= 0 {
		return fmt.Errorf("either MaxEntryCount or MaxMemoryBytes must be positive")
	}

	if config.Shards <= 0 {
		return fmt.Errorf("Shards must be positive")
	}

	switch config.EvictionPolicy {
	case "lru", "lfu", "fifo", "random":
		// Valid eviction policies
	default:
		return fmt.Errorf("unsupported eviction policy: %s", config.EvictionPolicy)
	}

	return nil
}
