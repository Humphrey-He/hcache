// Package codec defines interfaces for serializing and deserializing
// cache values.
package codec

// Codec defines the interface for encoding and decoding cache values.
// Implementations of this interface can be used to customize how values
// are serialized and deserialized in the cache.
type Codec interface {
	// Marshal serializes a value into bytes.
	// The value can be of any type that the codec supports.
	//
	// Parameters:
	//   - value: The value to serialize
	//
	// Returns:
	//   - []byte: The serialized bytes
	//   - error: An error if serialization fails
	Marshal(value interface{}) ([]byte, error)

	// Unmarshal deserializes bytes into a value.
	// The value parameter should be a pointer to the target type.
	//
	// Parameters:
	//   - data: The bytes to deserialize
	//   - value: A pointer to the target value
	//
	// Returns:
	//   - error: An error if deserialization fails
	Unmarshal(data []byte, value interface{}) error

	// Name returns the name of this codec.
	// This is useful for identification and debugging.
	//
	// Returns:
	//   - string: The codec name
	Name() string
}

// Compressor defines the interface for compressing and decompressing data.
// Implementations can be used to reduce the memory footprint of cached values.
type Compressor interface {
	// Compress compresses the input data.
	//
	// Parameters:
	//   - data: The data to compress
	//
	// Returns:
	//   - []byte: The compressed data
	//   - error: An error if compression fails
	Compress(data []byte) ([]byte, error)

	// Decompress decompresses the input data.
	//
	// Parameters:
	//   - data: The data to decompress
	//
	// Returns:
	//   - []byte: The decompressed data
	//   - error: An error if decompression fails
	Decompress(data []byte) ([]byte, error)

	// Name returns the name of this compressor.
	//
	// Returns:
	//   - string: The compressor name
	Name() string
}
