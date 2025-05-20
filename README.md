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

## Supported Formats

The package currently supports the following serialization formats:

- **JSON**: Standard JSON serialization
- **Gob**: Go's built-in binary serialization
- **MessagePack**: Efficient binary serialization format

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
   - MessagePack for efficient binary serialization and better type preservation

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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.