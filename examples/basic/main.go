package main

import (
	"fmt"
	"log"

	"github.com/MichaelAJay/go-serializer"
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
	serializer := &serializer.JSONSerializer{}

	// Serialize the person
	bytes, err := serializer.Serialize(person)
	if err != nil {
		log.Fatalf("Failed to serialize: %v", err)
	}

	fmt.Printf("Serialized JSON: %s\n", string(bytes))

	// Deserialize back to a person
	var result Person
	err = serializer.Deserialize(bytes, &result)
	if err != nil {
		log.Fatalf("Failed to deserialize: %v", err)
	}

	fmt.Printf("Deserialized person: %+v\n", result)
}
