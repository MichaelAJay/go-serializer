package serializer_test

import (
	"fmt"
	"io"
	"testing"

	"github.com/MichaelAJay/go-serializer"
)

// mockSerializer implements only the Serializer interface (not StringDeserializer)
type mockSerializer struct{}

func (m *mockSerializer) Serialize(v any) ([]byte, error) {
	return []byte("mock-data"), nil
}

func (m *mockSerializer) Deserialize(data []byte, v any) error {
	return nil
}

func (m *mockSerializer) SerializeTo(w io.Writer, v any) error {
	return nil
}

func (m *mockSerializer) DeserializeFrom(r io.Reader, v any) error {
	return nil
}

func (m *mockSerializer) ContentType() string {
	return "application/mock"
}

// mockStringSerializer implements both Serializer and StringDeserializer
type mockStringSerializer struct {
	*mockSerializer
}

func (m *mockStringSerializer) DeserializeString(data string, v any) error {
	if v == nil {
		return fmt.Errorf("nil target pointer")
	}
	return nil
}

// TestInterfaceDetection tests that StringDeserializer interface detection works correctly
func TestInterfaceDetection(t *testing.T) {
	tests := []struct {
		name                  string
		serializer            serializer.Serializer
		implementsStringDeser bool
	}{
		{
			name:                  "JSON_implements_StringDeserializer",
			serializer:            serializer.NewJSONSerializer(maxBufferSize),
			implementsStringDeser: true,
		},
		{
			name:                  "MsgPack_implements_StringDeserializer",
			serializer:            serializer.NewMsgpackSerializer(),
			implementsStringDeser: true,
		},
		{
			name:                  "Gob_implements_StringDeserializer",
			serializer:            serializer.NewGobSerializer(),
			implementsStringDeser: true,
		},
		{
			name:                  "Mock_does_not_implement_StringDeserializer",
			serializer:            &mockSerializer{},
			implementsStringDeser: false,
		},
		{
			name:                  "MockString_implements_StringDeserializer",
			serializer:            &mockStringSerializer{mockSerializer: &mockSerializer{}},
			implementsStringDeser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stringDeser, ok := tt.serializer.(serializer.StringDeserializer)

			if tt.implementsStringDeser {
				if !ok {
					t.Errorf("Expected serializer to implement StringDeserializer, but it doesn't")
				}
				if stringDeser == nil {
					t.Errorf("Expected non-nil StringDeserializer, got nil")
				}
			} else {
				if ok {
					t.Errorf("Expected serializer to NOT implement StringDeserializer, but it does")
				}
				if stringDeser != nil {
					t.Errorf("Expected nil StringDeserializer, got %v", stringDeser)
				}
			}
		})
	}
}

// TestTypeAssertionSafety tests that type assertion is safe and doesn't panic
func TestTypeAssertionSafety(t *testing.T) {
	serializers := []serializer.Serializer{
		serializer.NewJSONSerializer(maxBufferSize),
		serializer.NewMsgpackSerializer(),
		serializer.NewGobSerializer(),
		&mockSerializer{},
		&mockStringSerializer{mockSerializer: &mockSerializer{}},
	}

	for _, s := range serializers {
		t.Run(s.ContentType(), func(t *testing.T) {
			// Test that type assertion doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Type assertion panicked: %v", r)
				}
			}()

			// Safe type assertion
			if stringDeser, ok := s.(serializer.StringDeserializer); ok {
				// If it implements StringDeserializer, test that we can call the method
				err := stringDeser.DeserializeString("test", nil)
				// We expect an error for nil target, but no panic
				if err == nil {
					t.Error("Expected error for nil target, got none")
				}
			}
		})
	}
}

// TestFallbackBehavior tests graceful fallback when StringDeserializer is not available
func TestFallbackBehavior(t *testing.T) {
	// Test data
	testData := "test string"

	// Test with serializers that implement StringDeserializer
	realSerializers := []serializer.Serializer{
		serializer.NewJSONSerializer(maxBufferSize),
		serializer.NewMsgpackSerializer(),
		serializer.NewGobSerializer(),
	}

	for _, s := range realSerializers {
		t.Run("WithStringDeser_"+s.ContentType(), func(t *testing.T) {
			// Serialize test data
			data, err := s.Serialize(testData)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Test StringDeserializer path
			if stringDeser, ok := s.(serializer.StringDeserializer); ok {
				var stringResult string
				err := stringDeser.DeserializeString(string(data), &stringResult)
				if err != nil {
					t.Fatalf("DeserializeString failed: %v", err)
				}

				// Test fallback path (traditional Deserialize)
				var byteResult string
				err = s.Deserialize(data, &byteResult)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}

				// Results should be identical
				if stringResult != byteResult {
					t.Errorf("StringDeserializer and Deserialize results differ: %q vs %q", stringResult, byteResult)
				}
			} else {
				t.Error("Expected serializer to implement StringDeserializer")
			}
		})
	}

	// Test with mock serializer that doesn't implement StringDeserializer
	t.Run("WithoutStringDeser", func(t *testing.T) {
		var mock serializer.Serializer = &mockSerializer{}

		// Should not implement StringDeserializer
		if _, ok := mock.(serializer.StringDeserializer); ok {
			t.Error("Mock serializer should not implement StringDeserializer")
		}

		// Should still be usable as regular Serializer
		data, err := mock.Serialize(testData)
		if err != nil {
			t.Fatalf("Serialize failed: %v", err)
		}

		var result string
		err = mock.Deserialize(data, &result)
		if err != nil {
			t.Fatalf("Deserialize failed: %v", err)
		}
	})
}

// TestRegistryWithStringDeserializer tests that registry works with StringDeserializer implementations
func TestRegistryWithStringDeserializer(t *testing.T) {
	registry := serializer.NewRegistry()

	// Register all serializers
	registry.Register("json", serializer.NewJSONSerializer(maxBufferSize))
	registry.Register("msgpack", serializer.NewMsgpackSerializer())
	registry.Register("gob", serializer.NewGobSerializer())
	registry.Register("mock", &mockSerializer{})
	registry.Register("mockstring", &mockStringSerializer{mockSerializer: &mockSerializer{}})

	formats := []serializer.Format{"json", "msgpack", "gob", "mock", "mockstring"}
	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			// Get serializer from registry
			s, ok := registry.Get(format)
			if !ok {
				t.Fatalf("Failed to get serializer for format %s", format)
			}

			// Test interface detection
			_, hasStringDeser := s.(serializer.StringDeserializer)

			switch format {
			case "json", "msgpack", "gob", "mockstring":
				if !hasStringDeser {
					t.Errorf("Expected format %s to implement StringDeserializer", format)
				}
			case "mock":
				if hasStringDeser {
					t.Errorf("Expected format %s to NOT implement StringDeserializer", format)
				}
			}

			// Test New method
			newS, err := registry.New(format)
			if err != nil {
				t.Fatalf("Failed to create new serializer for format %s: %v", format, err)
			}

			// New serializer should have same interface capabilities
			_, newHasStringDeser := newS.(serializer.StringDeserializer)
			if hasStringDeser != newHasStringDeser {
				t.Errorf("New serializer has different StringDeserializer capability than original")
			}
		})
	}
}

// TestConcurrentInterfaceDetection tests interface detection under concurrent access
func TestConcurrentInterfaceDetection(t *testing.T) {
	serializers := []serializer.Serializer{
		serializer.NewJSONSerializer(maxBufferSize),
		serializer.NewMsgpackSerializer(),
		serializer.NewGobSerializer(),
	}

	const numGoroutines = 10
	const numIterations = 100

	for _, s := range serializers {
		t.Run(s.ContentType(), func(t *testing.T) {
			done := make(chan bool, numGoroutines)

			// Start multiple goroutines doing interface detection
			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("Goroutine panicked: %v", r)
						}
						done <- true
					}()

					for j := 0; j < numIterations; j++ {
						// Type assertion should always work consistently
						stringDeser, ok := s.(serializer.StringDeserializer)
						if !ok {
							t.Errorf("Expected serializer to implement StringDeserializer")
							return
						}
						if stringDeser == nil {
							t.Errorf("Expected non-nil StringDeserializer")
							return
						}
					}
				}()
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				<-done
			}
		})
	}
}

// TestNilSerializerHandling tests behavior with nil serializers
func TestNilSerializerHandling(t *testing.T) {
	var nilSerializer serializer.Serializer

	// Test that nil serializer doesn't implement StringDeserializer
	stringDeser, ok := nilSerializer.(serializer.StringDeserializer)
	if ok {
		t.Error("nil serializer should not implement StringDeserializer")
	}
	if stringDeser != nil {
		t.Error("StringDeserializer should be nil for nil serializer")
	}
}
