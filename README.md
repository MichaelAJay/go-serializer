# Go Serializer

A Go library for uniform serialization and deserialization across different formats.

[![Go Report Card](https://goreportcard.com/badge/github.com/MichaelAJay/go-serializer)](https://goreportcard.com/report/github.com/MichaelAJay/go-serializer)
[![GoDoc](https://godoc.org/github.com/MichaelAJay/go-serializer?status.svg)](https://godoc.org/github.com/MichaelAJay/go-serializer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Current version: v0.1.0

## Features

- Support for multiple serialization formats:
  - JSON
  - Gob
  - MessagePack
- Uniform serialization behavior across all formats
- Type preservation during serialization/deserialization
- Cross-format compatibility
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
    registry.Register("json", serializer.NewJSONSerializer())
    registry.Register("gob", serializer.NewGobSerializer())
    registry.Register("msgpack", serializer.NewMsgpackSerializer())

    // Get a serializer
    jsonSerializer := registry.Get("json")

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

    // Get the type of serialized data
    valueType, err := jsonSerializer.GetType(bytes)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Type: %s\n", valueType)

    // Deserialize data
    var result map[string]any
    err = jsonSerializer.Deserialize(bytes, &result)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Result: %+v\n", result)
}
```

### Uniform Serialization

The library ensures uniform serialization behavior across all supported formats:

1. **Type Preservation**: All serializers preserve type information during serialization and deserialization.
2. **Cross-Format Compatibility**: Data serialized with one format can be deserialized with another.
3. **Consistent Type Handling**: All serializers handle types consistently:
   - Basic types (string, int, float, bool) are preserved exactly
   - Slices and maps maintain their structure and element types
   - Structs preserve their field types and values
   - Interface{} values are handled uniformly

Example of cross-format compatibility:

```go
// Serialize with JSON
jsonBytes, _ := jsonSerializer.Serialize(data)

// Deserialize with MessagePack
var result map[string]any
msgpackSerializer.Deserialize(jsonBytes, &result)
// result will match the original data exactly
```

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
registry.Register("json", serializer.NewJSONSerializer())
registry.Register("gob", serializer.NewGobSerializer())
registry.Register("msgpack", serializer.NewMsgpackSerializer())

// Get a serializer
jsonSerializer := registry.Get("json")

// Create a new serializer
newSerializer := registry.New("json")
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