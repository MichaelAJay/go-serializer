package main

import (
	"encoding/gob"
	"fmt"
	"time"

	"github.com/MichaelAJay/go-serializer"
)

const (
	maxBufferSize = 32 * 1024
)

func main() {
	// Register types with gob to allow serialization
	gob.Register(time.Time{})
	gob.Register(map[string]int{})
	gob.Register(map[string]any{})

	// Create a registry
	registry := serializer.NewRegistry()

	// Register serializers
	registry.Register(serializer.JSON, serializer.NewJSONSerializer(maxBufferSize))
	registry.Register(serializer.Binary, serializer.NewGobSerializer())
	registry.Register(serializer.Msgpack, serializer.NewMsgpackSerializer())

	// Test data with various types
	data := map[string]any{
		"string":    "test",
		"int":       42,
		"float":     3.14,
		"bool":      true,
		"time":      time.Now(),
		"slice":     []string{"a", "b", "c"},
		"map":       map[string]int{"x": 1, "y": 2},
		"interface": "interface value",
	}

	// Test each serializer
	for _, format := range []serializer.Format{serializer.JSON, serializer.Binary, serializer.Msgpack} {
		fmt.Printf("\nTesting %s serializer:\n", format)
		ser, ok := registry.Get(format)
		if !ok {
			fmt.Printf("Serializer not found: %s\n", format)
			continue
		}

		// Serialize the data
		bytes, err := ser.Serialize(data)
		if err != nil {
			fmt.Printf("Serialization error: %v\n", err)
			continue
		}

		fmt.Printf("Content-Type: %s\n", ser.ContentType())
		fmt.Printf("Serialized size: %d bytes\n", len(bytes))

		// Deserialize the data
		var result map[string]any
		err = ser.Deserialize(bytes, &result)
		if err != nil {
			fmt.Printf("Deserialization error: %v\n", err)
			continue
		}

		// Print the result
		fmt.Printf("Result contains %d keys\n", len(result))

		// Compare a few values
		if s, ok := result["string"].(string); ok {
			fmt.Printf("String value: %s\n", s)
		}

		// Print numeric values - note how JSON always returns float64
		switch format {
		case serializer.JSON:
			if n, ok := result["int"].(float64); ok {
				fmt.Printf("JSON numeric value (stored as float64): %v\n", n)
			}
		case serializer.Binary, serializer.Msgpack:
			if n, ok := result["int"].(int); ok {
				fmt.Printf("Numeric value (preserved as int): %v\n", n)
			}
		}
	}

	// Using serializers from default registry
	fmt.Println("\nUsing default registry:")
	jsonSerializer, _ := serializer.DefaultRegistry.Get(serializer.JSON)
	msgpackSerializer, _ := serializer.DefaultRegistry.Get(serializer.Msgpack)

	fmt.Printf("JSON content type: %s\n", jsonSerializer.ContentType())
	fmt.Printf("MsgPack content type: %s\n", msgpackSerializer.ContentType())
}
