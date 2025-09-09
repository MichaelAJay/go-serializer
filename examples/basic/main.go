package main

import (
	"fmt"
	"log"

	"github.com/MichaelAJay/go-serializer"
)

const (
	maxBufferSize = 32 * 1024
)

type Person struct {
	Name    string   `json:"name"`
	Age     int      `json:"age"`
	Hobbies []string `json:"hobbies"`
}

func main() {
	// Create a person
	person := Person{
		Name:    "John Doe",
		Age:     30,
		Hobbies: []string{"reading", "gaming", "coding"},
	}

	// Create a JSON serializer
	jsonSerializer := serializer.NewJSONSerializer(maxBufferSize)

	// Serialize the person
	bytes, err := jsonSerializer.Serialize(person)
	if err != nil {
		log.Fatalf("Failed to serialize: %v", err)
	}

	fmt.Printf("Serialized JSON: %s\n", string(bytes))

	// Deserialize back to a person
	var result Person
	err = jsonSerializer.Deserialize(bytes, &result)
	if err != nil {
		log.Fatalf("Failed to deserialize: %v", err)
	}

	fmt.Printf("Deserialized person: %+v\n", result)

	// Try other serializers
	fmt.Println("\nTrying MessagePack serializer:")
	msgpackSerializer := serializer.NewMsgpackSerializer()

	// Serialize to MessagePack
	msgpackBytes, err := msgpackSerializer.Serialize(person)
	if err != nil {
		log.Fatalf("Failed to serialize with MessagePack: %v", err)
	}
	fmt.Printf("MessagePack serialized size: %d bytes\n", len(msgpackBytes))

	// Deserialize from MessagePack
	var msgpackResult Person
	err = msgpackSerializer.Deserialize(msgpackBytes, &msgpackResult)
	if err != nil {
		log.Fatalf("Failed to deserialize with MessagePack: %v", err)
	}

	fmt.Printf("MessagePack deserialized person: %+v\n", msgpackResult)
}
