package main

import (
	"fmt"
	"log"

	"github.com/MichaelAJay/go-serializer"
)

func main() {
	// Create a registry
	registry := serializer.NewRegistry()

	// Register all available serializers
	registry.Register(serializer.JSON, serializer.NewJSONSerializer())
	registry.Register(serializer.Binary, serializer.NewGobSerializer())
	registry.Register(serializer.Msgpack, serializer.NewMsgpackSerializer())

	// Data to serialize
	data := map[string]any{
		"name":    "Alice",
		"age":     25,
		"active":  true,
		"scores":  []int{95, 87, 92},
		"address": map[string]string{"city": "New York", "country": "USA"},
	}

	// Try each serializer
	formats := []serializer.Format{
		serializer.JSON,
		serializer.Binary,
		serializer.Msgpack,
	}

	for _, format := range formats {
		// Get serializer
		ser, err := registry.New(format)
		if err != nil {
			log.Printf("Failed to get serializer for %s: %v", format, err)
			continue
		}

		// Serialize
		bytes, err := ser.Serialize(data)
		if err != nil {
			log.Printf("Failed to serialize with %s: %v", format, err)
			continue
		}

		// Get type of serialized data
		valueType, err := ser.GetType(bytes)
		if err != nil {
			log.Printf("Failed to get type with %s: %v", format, err)
			continue
		}

		// Deserialize
		var result map[string]any
		err = ser.Deserialize(bytes, &result)
		if err != nil {
			log.Printf("Failed to deserialize with %s: %v", format, err)
			continue
		}

		fmt.Printf("\nFormat: %s\n", format)
		fmt.Printf("Content-Type: %s\n", ser.ContentType())
		fmt.Printf("Type: %s\n", valueType)
		fmt.Printf("Serialized size: %d bytes\n", len(bytes))
		fmt.Printf("Deserialized data: %+v\n", result)
	}
}
