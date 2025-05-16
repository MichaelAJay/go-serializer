package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/MichaelAJay/go-serializer"
)

type LogEntry struct {
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata"`
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func main() {
	// Create some log entries
	entries := []LogEntry{
		{
			Timestamp: "2024-03-20T10:00:00Z",
			Level:     "INFO",
			Message:   "Application started",
			Metadata: map[string]string{
				"version": "1.0.0",
				"env":     "production",
			},
		},
		{
			Timestamp: "2024-03-20T10:01:00Z",
			Level:     "WARN",
			Message:   "High memory usage",
			Metadata: map[string]string{
				"memory_usage": "85%",
				"threshold":    "80%",
			},
		},
	}

	// Create a buffer for streaming
	var buf bytes.Buffer

	// Create a MessagePack serializer (good for binary data)
	msgpackSerializer := serializer.NewMsgpackSerializer()

	// Serialize entries to stream
	err := msgpackSerializer.SerializeTo(&buf, entries)
	if err != nil {
		log.Fatalf("Failed to serialize to stream: %v", err)
	}

	fmt.Printf("MessagePack serialized size: %d bytes\n", buf.Len())

	// Deserialize from stream
	var result []LogEntry
	err = msgpackSerializer.DeserializeFrom(&buf, &result)
	if err != nil {
		log.Fatalf("Failed to deserialize from stream: %v", err)
	}

	// Print deserialized entries
	fmt.Printf("Deserialized %d log entries\n", len(result))
	for i, entry := range result {
		fmt.Printf("\nEntry %d:\n", i+1)
		fmt.Printf("  Timestamp: %s\n", entry.Timestamp)
		fmt.Printf("  Level: %s\n", entry.Level)
		fmt.Printf("  Message: %s\n", entry.Message)
		fmt.Printf("  Metadata: %v\n", entry.Metadata)
	}

	// Compare with JSON serialization
	fmt.Println("\nComparing with JSON serialization:")
	jsonSerializer := serializer.NewJSONSerializer()

	// Reset buffer
	buf.Reset()

	// Serialize to JSON stream
	err = jsonSerializer.SerializeTo(&buf, entries)
	if err != nil {
		log.Fatalf("Failed to serialize to JSON stream: %v", err)
	}

	fmt.Printf("JSON serialized size: %d bytes\n", buf.Len())
	fmt.Printf("JSON content (first 100 chars): %s\n", string(buf.Bytes()[:min(100, buf.Len())])+"...")
}
