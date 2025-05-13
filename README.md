# Go Serializer

A flexible and extensible serialization package for Go applications. This package provides a unified interface for different serialization formats with support for both byte-based and streaming operations.

[![Go Report Card](https://goreportcard.com/badge/github.com/MichaelAJay/go-serializer)](https://goreportcard.com/report/github.com/MichaelAJay/go-serializer)
[![GoDoc](https://godoc.org/github.com/MichaelAJay/go-serializer?status.svg)](https://godoc.org/github.com/MichaelAJay/go-serializer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Current version: v0.1.0

## Features

- Multiple serialization formats (JSON, Gob, MessagePack)
- Unified interface for all serializers
- Support for both byte-based and streaming operations
- Registry system for easy serializer management
- Thread-safe operations
- Proper error handling
- Content type support for HTTP operations
- Version information and management

## Installation

```bash
go get github.com/MichaelAJay/go-serializer
```

## Quick Start

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
    registry.Register(serializer.JSON, &serializer.JSONSerializer{})
    registry.Register(serializer.Binary, &serializer.GobSerializer{})
    registry.Register(serializer.Msgpack, &serializer.MsgPackSerializer{})

    // Get a serializer
    jsonSerializer, _ := registry.Get(serializer.JSON)

    // Serialize data
    data := map[string]interface{}{
        "name": "John",
        "age":  30,
    }
    bytes, err := jsonSerializer.Serialize(data)
    if err != nil {
        panic(err)
    }

    // Deserialize data
    var result map[string]interface{}
    err = jsonSerializer.Deserialize(bytes, &result)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Deserialized: %+v\n", result)
}
```

## Examples

The package includes several examples demonstrating different use cases:

1. **Basic Usage** (`examples/basic/main.go`):
   - Simple serialization of a struct
   - JSON serialization and deserialization
   - Error handling

2. **Registry Usage** (`examples/registry/main.go`):
   - Managing multiple serializers
   - Comparing different formats
   - Content type handling

3. **Streaming Operations** (`examples/streaming/main.go`):
   - Streaming serialization
   - Working with binary data
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

## Usage

### Basic Serialization

```go
// Create a serializer
serializer := &serializer.JSONSerializer{}

// Serialize data
data := []string{"a", "b", "c"}
bytes, err := serializer.Serialize(data)
if err != nil {
    panic(err)
}

// Deserialize data
var result []string
err = serializer.Deserialize(bytes, &result)
if err != nil {
    panic(err)
}
```

### Streaming Operations

```go
// Create a buffer for streaming
var buf bytes.Buffer

// Serialize to stream
err := serializer.SerializeTo(&buf, data)
if err != nil {
    panic(err)
}

// Deserialize from stream
var result []string
err = serializer.DeserializeFrom(&buf, &result)
if err != nil {
    panic(err)
}
```

### Using the Registry

```go
// Create and configure registry
registry := serializer.NewRegistry()
registry.Register(serializer.JSON, &serializer.JSONSerializer{})
registry.Register(serializer.Binary, &serializer.GobSerializer{})
registry.Register(serializer.Msgpack, &serializer.MsgPackSerializer{})

// Get a serializer by format
serializer, ok := registry.Get(serializer.JSON)
if !ok {
    panic("JSON serializer not found")
}

// Create a new serializer instance
serializer, err := registry.New(serializer.JSON)
if err != nil {
    panic(err)
}
```

## Content Types

Each serializer provides its content type for HTTP operations:

- JSON: `application/json`
- Gob: `application/octet-stream`
- MessagePack: `application/msgpack`

## Error Handling

The package provides comprehensive error handling:

- Nil value checks
- Invalid data validation
- Stream operation errors
- Registry errors

## Best Practices

1. **Format Selection**: Choose the appropriate format for your use case:
   - JSON for human-readable data and web APIs
   - Gob for Go-specific applications
   - MessagePack for efficient binary serialization

2. **Error Handling**: Always check for errors in serialization operations

3. **Type Safety**: Use strongly-typed structs when possible

4. **Streaming**: Use streaming operations for large data sets

5. **Registry**: Use the registry for managing multiple serializers

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.