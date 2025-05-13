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
	serializer := &serializer.MsgPackSerializer{}

	// Serialize entries to stream
	err := serializer.SerializeTo(&buf, entries)
	if err != nil {
		log.Fatalf("Failed to serialize to stream: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", buf.Len())

	// Deserialize from stream
	var result []LogEntry
	err = serializer.DeserializeFrom(&buf, &result)
	if err != nil {
		log.Fatalf("Failed to deserialize from stream: %v", err)
	}

	// Print deserialized entries
	for _, entry := range result {
		fmt.Printf("\nEntry:\n")
		fmt.Printf("  Timestamp: %s\n", entry.Timestamp)
		fmt.Printf("  Level: %s\n", entry.Level)
		fmt.Printf("  Message: %s\n", entry.Message)
		fmt.Printf("  Metadata: %v\n", entry.Metadata)
	}
}
