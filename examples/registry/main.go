package main

import (
	"fmt"
	"time"

	"github.com/MichaelAJay/go-serializer"
)

func main() {
	// Create a registry
	registry := serializer.NewRegistry()

	// Register serializers
	registry.Register(serializer.JSON, serializer.NewJSONSerializer())
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

		// Get the type information
		valueType, err := ser.GetType(bytes)
		if err != nil {
			fmt.Printf("GetType error: %v\n", err)
			continue
		}
		fmt.Printf("Content-Type: %s\n", ser.ContentType())
		fmt.Printf("Type: %s\n", valueType)

		// Deserialize the data
		var result map[string]any
		err = ser.Deserialize(bytes, &result)
		if err != nil {
			fmt.Printf("Deserialization error: %v\n", err)
			continue
		}

		// Print the result
		fmt.Printf("Result: %+v\n", result)

		// Test cross-format compatibility
		if format != serializer.JSON {
			fmt.Printf("\nTesting cross-format compatibility (json -> %s):\n", format)
			jsonSer, ok := registry.Get(serializer.JSON)
			if !ok {
				fmt.Printf("JSON serializer not found\n")
				continue
			}
			jsonBytes, err := jsonSer.Serialize(data)
			if err != nil {
				fmt.Printf("JSON serialization error: %v\n", err)
				continue
			}
			var crossResult map[string]any
			err = ser.Deserialize(jsonBytes, &crossResult)
			if err != nil {
				fmt.Printf("Cross-format deserialization error: %v\n", err)
				continue
			}
			fmt.Printf("Cross-format result: %+v\n", crossResult)
		}
	}
}
