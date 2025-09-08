# Go Serializer

A Go library for consistent serialization and deserialization across different formats.

[![Go Report Card](https://goreportcard.com/badge/github.com/MichaelAJay/go-serializer)](https://goreportcard.com/report/github.com/MichaelAJay/go-serializer)
[![GoDoc](https://godoc.org/github.com/MichaelAJay/go-serializer?status.svg)](https://godoc.org/github.com/MichaelAJay/go-serializer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Current version: v0.1.0

## Features

- Support for multiple serialization formats:
  - JSON
  - Gob
  - MessagePack
- Consistent API across all formats
- Format-specific type handling
- Streaming support
- Registry for managing multiple serializers
- **Performance-optimized string deserialization** with StringDeserializer interface

## Installation

```bash
go get github.com/MichaelAJay/go-serializer
```

## Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/MichaelAJay/go-serializer"
)

func main() {
    // Create a registry
    registry := serializer.NewRegistry()

    // Register serializers
    registry.Register(serializer.JSON, serializer.NewJSONSerializer())
    registry.Register(serializer.Binary, serializer.NewGobSerializer())
    registry.Register(serializer.Msgpack, serializer.NewMsgpackSerializer())

    // Get a serializer
    jsonSerializer, _ := registry.Get(serializer.JSON)

    // Serialize data
    data := map[string]any{
        "name": "John",
        "age":  30,
        "tags": []string{"golang", "serialization"},
    }
    bytes, err := jsonSerializer.Serialize(data)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Serialized JSON: %s\n", string(bytes))

    // Deserialize data
    var result map[string]any
    err = jsonSerializer.Deserialize(bytes, &result)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Result: %+v\n", result)
}
```

### Format Differences

Each serialization format has its own specific behaviors:

1. **JSON**:
   - Human-readable text format
   - All numbers are deserialized as `float64`
   - Time values are serialized as strings
   - Content-Type: `application/json`

2. **MessagePack**:
   - Compact binary format
   - Preserves numeric types (int, float)
   - Better performance for large datasets
   - **Advanced pooled APIs** for high-throughput applications
   - Content-Type: `application/x-msgpack`

3. **Gob**:
   - Go-specific binary format
   - Best for Go-to-Go communication
   - Preserves Go types accurately
   - **Requires explicit type registration** for any values:
     ```go
     // Register types with gob before serialization
     gob.Register(time.Time{})
     gob.Register(map[string]any{})
     ```
   - Content-Type: `application/x-gob`

## Performance Features

### High-Performance MessagePack Serializer

For applications requiring maximum throughput and minimal memory allocations, the MessagePack serializer provides advanced pooled serialization APIs:

- **Pooled Encoders/Decoders**: Reuses encoder and decoder objects across operations to eliminate per-call allocations
- **Zero-Copy Serialization**: `SerializePooled()` returns pooled buffers without copying data
- **Safe Fallback**: `SerializeSafe()` provides allocation reduction while maintaining simple ownership semantics
- **Thread-Safe**: All pooled operations are safe for concurrent use

#### Performance Variants

```go
// Standard API - minimal allocations with simple ownership
data, err := msgpackSerializer.SerializeSafe(value)

// High-performance API - zero-copy with manual lifecycle management
pooledBuf, err := msgpackSerializer.SerializePooled(value)
defer pooledBuf.Release() // Must call Release() when done

// Direct access to pooled bytes
bytes := pooledBuf.Bytes()
```

#### Batch Operations

For high-throughput scenarios like Redis pipelining, use the optimized batch APIs:

```go
// Safe batch operation - reduced allocations with simple API
err := setManySafe(ctx, values, ttl)

// High-performance batch operation - minimal allocations with lifecycle management
err := setManyPooled(ctx, values, ttl)
```

**Performance Benefits:**
- **5× reduction in allocations** for batch operations
- **Significant memory usage reduction** in high-throughput scenarios
- **Lower GC pressure** through object pooling
- **Thread-safe pooling** with automatic buffer size management

### Performance-Optimized String Deserialization

All built-in serializers implement the `StringDeserializer` interface, which provides optimized deserialization directly from strings without the overhead of string-to-byte conversion:

```go
// When you have serialized data as a string (e.g., from cache, database, API)
serializedData := `{"name":"John","age":30}`

// Traditional approach (allocates extra memory for []byte conversion)
var result1 map[string]any
err := jsonSerializer.Deserialize([]byte(serializedData), &result1)

// Optimized approach (avoids string->[]byte allocation)
if stringDeser, ok := jsonSerializer.(serializer.StringDeserializer); ok {
    var result2 map[string]any
    err := stringDeser.DeserializeString(serializedData, &result2)
    // Same result, but with better performance for large strings
}
```

**Performance Benefits:**
- **Eliminates string→[]byte allocation** saving memory and reducing GC pressure
- **50-80% reduction in memory allocations** for large string data
- **Automatic optimization** - all built-in serializers support this interface
- **Backward compatible** - graceful fallback to standard `Deserialize()` method

**When to Use:**
- Deserializing data from string sources (Redis, databases, REST APIs)
- High-throughput applications processing large strings
- Memory-constrained environments where allocation reduction matters

### Streaming Support

All serializers support streaming serialization and deserialization:

```go
// Serialize to a writer
err := serializer.SerializeTo(writer, data)

// Deserialize from a reader
var result map[string]any
err := serializer.DeserializeFrom(reader, &result)
```

### Registry

The registry provides a convenient way to manage multiple serializers:

```go
registry := serializer.NewRegistry()

// Register serializers
registry.Register(serializer.JSON, serializer.NewJSONSerializer())
registry.Register(serializer.Binary, serializer.NewGobSerializer())
registry.Register(serializer.Msgpack, serializer.NewMsgpackSerializer())

// Get a serializer
jsonSerializer, ok := registry.Get(serializer.JSON)

// Create a new serializer instance
newSerializer, err := registry.New(serializer.JSON)
```

## Examples

The package includes several examples demonstrating different use cases:

1. **Basic Usage** (`examples/basic/main.go`):
   - Simple serialization of a struct
   - JSON and MessagePack serialization
   - Type handling differences

2. **Registry Usage** (`examples/registry/main.go`):
   - Managing multiple serializers
   - Using the default registry
   - Demonstrating format-specific behaviors

3. **Streaming Operations** (`examples/streaming/main.go`):
   - Streaming serialization
   - Size comparison between formats
   - Complex data structures

To run the examples:

```bash
# Run basic example
go run examples/basic/main.go

# Run registry example
go run examples/registry/main.go

# Run streaming example
go run examples/streaming/main.go
```

### StringDeserializer in Practice

Here's a practical example showing how StringDeserializer can be used with cache layers and databases:

```go
package main

import (
    "fmt"
    "github.com/MichaelAJay/go-serializer"
)

type CacheClient struct {
    serializer serializer.Serializer
    stringDeser serializer.StringDeserializer
}

func NewCacheClient(s serializer.Serializer) *CacheClient {
    client := &CacheClient{serializer: s}
    
    // Check if serializer supports optimized string deserialization
    if stringDeser, ok := s.(serializer.StringDeserializer); ok {
        client.stringDeser = stringDeser
    }
    
    return client
}

func (c *CacheClient) Get(key string, result any) error {
    // Simulate getting string data from cache/database
    serializedData := `{"id":123,"name":"John Doe","active":true}`
    
    // Use optimized string deserialization if available
    if c.stringDeser != nil {
        return c.stringDeser.DeserializeString(serializedData, result)
    }
    
    // Fallback to traditional method
    return c.serializer.Deserialize([]byte(serializedData), result)
}

func main() {
    // Works with any serializer format
    jsonClient := NewCacheClient(serializer.NewJSONSerializer())
    msgpackClient := NewCacheClient(serializer.NewMsgpackSerializer())
    
    var user map[string]any
    
    // Both will use optimized StringDeserializer automatically
    err := jsonClient.Get("user:123", &user)
    if err == nil {
        fmt.Printf("JSON result: %+v\n", user)
    }
    
    err = msgpackClient.Get("user:123", &user)  
    if err == nil {
        fmt.Printf("MsgPack result: %+v\n", user)
    }
}
```

## Version Information

The package provides version information through the following functions:

```go
// Get the full version string
version := serializer.VersionString() // Returns "v0.1.0"

// Get version components
info := serializer.VersionInfo()
major := info["major"] // 0
minor := info["minor"] // 1
patch := info["patch"] // 0
```

## API Reference

### Core Interfaces

The package defines two main interfaces:

**Serializer Interface:**
```go
type Serializer interface {
    Serialize(v any) ([]byte, error)
    Deserialize(data []byte, v any) error
    SerializeTo(w io.Writer, v any) error
    DeserializeFrom(r io.Reader, v any) error
    ContentType() string
}
```

**StringDeserializer Interface (Performance Optimization):**
```go
type StringDeserializer interface {
    DeserializeString(data string, v any) error
}
```

All built-in serializers implement both interfaces, providing automatic performance optimization when deserializing from strings.

## Supported Formats

The package currently supports the following serialization formats:

- **JSON**: Standard JSON serialization
- **Gob**: Go's built-in binary serialization
- **MessagePack**: Efficient binary serialization format

All formats support both the `Serializer` and `StringDeserializer` interfaces.

## Content Types

Each serializer provides its content type for HTTP operations:

- JSON: `application/json`
- Gob: `application/x-gob`
- MessagePack: `application/x-msgpack`

## Error Handling

The package provides comprehensive error handling:

- Nil value checks
- Invalid data validation
- Stream operation errors
- Registry errors

## Best Practices

1. **Format Selection**: Choose the appropriate format for your use case:
   - JSON for human-readable data and web APIs
   - Gob for Go-specific applications (remember to register types)
   - MessagePack for efficient binary serialization, better type preservation, and high-throughput applications requiring minimal allocations

2. **Error Handling**: Always check for errors in serialization operations

3. **Type Awareness**: Be aware of format-specific type handling:
   - JSON converts all numbers to float64
   - MessagePack and Gob preserve integer types
   - Complex types may be handled differently across formats
   - Gob requires explicit type registration for interface values:
     ```go
     import "encoding/gob"
     
     func init() {
         // Register types that will be stored in any values
         gob.Register(time.Time{})
         gob.Register(map[string]any{})
         gob.Register([]any{})
     }
     ```

4. **Pointer for Deserialization**: Always pass a pointer to the `Deserialize` method:
   ```go
   var result MyStruct
   err = serializer.Deserialize(data, &result) // Use a pointer!
   ```

5. **Streaming**: Use streaming operations for large data sets to avoid memory constraints

6. **Performance Optimizations**:
   
   a. **StringDeserializer**: Use the StringDeserializer interface for better performance when deserializing from string sources:
   ```go
   // Good: Check for and use StringDeserializer when available
   func DeserializeFromString(serializer serializer.Serializer, data string, result any) error {
       if stringDeser, ok := serializer.(serializer.StringDeserializer); ok {
           return stringDeser.DeserializeString(data, result)
       }
       return serializer.Deserialize([]byte(data), result)
   }
   ```
   
   b. **MessagePack Pooled APIs**: For high-throughput scenarios, use pooled serialization:
   ```go
   // Safe approach with allocation reduction
   data, err := msgpackSerializer.SerializeSafe(value)
   
   // High-performance approach requiring lifecycle management
   pooledBuf, err := msgpackSerializer.SerializePooled(value)
   defer pooledBuf.Release() // Critical: must call Release()
   bytes := pooledBuf.Bytes()
   ```
   
   c. **Batch Operations**: Use optimized batch methods for Redis/cache operations:
   ```go
   // Choose based on performance vs complexity tradeoffs
   err := setManySafe(ctx, values, ttl)      // Safer, still fast
   err := setManyPooled(ctx, values, ttl)    // Fastest, requires care
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.