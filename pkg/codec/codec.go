// Package codec provides interfaces and implementations for data serialization
// and deserialization used by the cache for storing and retrieving values.
// It offers various encoding formats including JSON, Gob, and string conversion.
//
// Package codec 提供用于缓存存储和检索值的数据序列化和反序列化接口及实现。
// 它提供各种编码格式，包括JSON、Gob和字符串转换。
package codec

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

// Codec defines the interface for encoding and decoding cache values.
// Implementations of this interface can be used to customize how values
// are serialized and deserialized in the cache.
//
// Codec 定义了编码和解码缓存值的接口。
// 此接口的实现可用于自定义如何在缓存中序列化和反序列化值。
type Codec interface {
	// Marshal serializes a value into bytes.
	// The value can be of any type that the codec supports.
	//
	// Marshal 将值序列化为字节。
	// 值可以是编解码器支持的任何类型。
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
	// Unmarshal 将字节反序列化为值。
	// value参数应该是目标类型的指针。
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
	// Name 返回此编解码器的名称。
	// 这对于标识和调试很有用。
	//
	// Returns:
	//   - string: The codec name
	Name() string
}

// JSONCodec implements Codec using JSON serialization.
// It provides efficient and human-readable encoding of values.
//
// JSONCodec 使用JSON序列化实现Codec。
// 它提供高效且人类可读的值编码。
type JSONCodec struct {
	// Pretty determines whether to use indented JSON encoding.
	// When true, the JSON output will be formatted with indentation.
	//
	// Pretty 决定是否使用缩进的JSON编码。
	// 当为true时，JSON输出将使用缩进格式化。
	Pretty bool
}

// Marshal serializes a value into JSON bytes.
// If Pretty is true, the output will be indented.
//
// Marshal 将值序列化为JSON字节。
// 如果Pretty为true，输出将带有缩进。
//
// Parameters:
//   - value: The value to serialize to JSON
//
// Returns:
//   - []byte: The JSON bytes
//   - error: An error if JSON serialization fails
func (c *JSONCodec) Marshal(value interface{}) ([]byte, error) {
	if c.Pretty {
		return json.MarshalIndent(value, "", "  ")
	}
	return json.Marshal(value)
}

// Unmarshal deserializes JSON bytes into a value.
// The value parameter must be a pointer to the target type.
//
// Unmarshal 将JSON字节反序列化为值。
// value参数必须是目标类型的指针。
//
// Parameters:
//   - data: The JSON bytes to deserialize
//   - value: A pointer to the target value
//
// Returns:
//   - error: An error if JSON deserialization fails
func (c *JSONCodec) Unmarshal(data []byte, value interface{}) error {
	return json.Unmarshal(data, value)
}

// Name returns the name of this codec.
//
// Name 返回此编解码器的名称。
//
// Returns:
//   - string: Always returns "json"
func (c *JSONCodec) Name() string {
	return "json"
}

// NewJSONCodec creates a new JSONCodec.
//
// NewJSONCodec 创建一个新的JSONCodec。
//
// Parameters:
//   - pretty: Whether to use indented JSON encoding
//
// Returns:
//   - *JSONCodec: A new JSON codec instance
func NewJSONCodec(pretty bool) *JSONCodec {
	return &JSONCodec{Pretty: pretty}
}

// GobCodec implements Codec using Gob serialization.
// Gob is a binary format optimized for Go types.
//
// GobCodec 使用Gob序列化实现Codec。
// Gob是为Go类型优化的二进制格式。
type GobCodec struct{}

// Marshal serializes a value into Gob bytes.
// The value must be encodable by the gob package.
//
// Marshal 将值序列化为Gob字节。
// 该值必须可由gob包编码。
//
// Parameters:
//   - value: The value to serialize
//
// Returns:
//   - []byte: The serialized bytes
//   - error: An error if Gob serialization fails
func (c *GobCodec) Marshal(value interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes Gob bytes into a value.
// The value parameter must be a pointer to the target type.
//
// Unmarshal 将Gob字节反序列化为值。
// value参数必须是目标类型的指针。
//
// Parameters:
//   - data: The Gob bytes to deserialize
//   - value: A pointer to the target value
//
// Returns:
//   - error: An error if Gob deserialization fails
func (c *GobCodec) Unmarshal(data []byte, value interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(value)
}

// Name returns the name of this codec.
//
// Name 返回此编解码器的名称。
//
// Returns:
//   - string: Always returns "gob"
func (c *GobCodec) Name() string {
	return "gob"
}

// NewGobCodec creates a new GobCodec.
//
// NewGobCodec 创建一个新的GobCodec。
//
// Returns:
//   - *GobCodec: A new Gob codec instance
func NewGobCodec() *GobCodec {
	return &GobCodec{}
}

// StringCodec implements Codec for string values.
// It provides simple conversion between strings and bytes.
//
// StringCodec 为字符串值实现Codec。
// 它提供字符串和字节之间的简单转换。
type StringCodec struct{}

// Marshal converts a string to bytes.
// The value must be a string or []byte.
//
// Marshal 将字符串转换为字节。
// 值必须是字符串或[]byte。
//
// Parameters:
//   - value: The string or []byte to convert
//
// Returns:
//   - []byte: The byte representation
//   - error: An error if the value is not a string or []byte
func (c *StringCodec) Marshal(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		return nil, fmt.Errorf("stringcodec: cannot marshal %T", value)
	}
}

// Unmarshal converts bytes to a string.
// The value parameter must be a pointer to a string or []byte.
//
// Unmarshal 将字节转换为字符串。
// value参数必须是指向字符串或[]byte的指针。
//
// Parameters:
//   - data: The bytes to convert
//   - value: A pointer to a string or []byte
//
// Returns:
//   - error: An error if value is not a pointer to a string or []byte
func (c *StringCodec) Unmarshal(data []byte, value interface{}) error {
	switch v := value.(type) {
	case *string:
		*v = string(data)
		return nil
	case *[]byte:
		*v = data
		return nil
	default:
		return fmt.Errorf("stringcodec: cannot unmarshal into %T", value)
	}
}

// Name returns the name of this codec.
//
// Name 返回此编解码器的名称。
//
// Returns:
//   - string: Always returns "string"
func (c *StringCodec) Name() string {
	return "string"
}

// NewStringCodec creates a new StringCodec.
//
// NewStringCodec 创建一个新的StringCodec。
//
// Returns:
//   - *StringCodec: A new string codec instance
func NewStringCodec() *StringCodec {
	return &StringCodec{}
}

// DefaultCodec returns the default codec (JSON).
// This is used when no specific codec is specified.
//
// DefaultCodec 返回默认编解码器（JSON）。
// 当未指定特定编解码器时使用。
//
// Returns:
//   - Codec: A default JSON codec instance
func DefaultCodec() Codec {
	return NewJSONCodec(false)
}

// GetCodec returns a codec by name.
// Supported names: "json", "gob", "string".
//
// GetCodec 通过名称返回编解码器。
// 支持的名称："json"、"gob"、"string"。
//
// Parameters:
//   - name: The codec name
//
// Returns:
//   - Codec: The requested codec
//   - error: An error if the codec name is unknown
func GetCodec(name string) (Codec, error) {
	switch name {
	case "json":
		return NewJSONCodec(false), nil
	case "gob":
		return NewGobCodec(), nil
	case "string":
		return NewStringCodec(), nil
	default:
		return nil, fmt.Errorf("unknown codec: %s", name)
	}
}
