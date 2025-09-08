package serializer

import (
	"testing"
)

func TestStringDeserializer(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name" msgpack:"name"`
		Value int    `json:"value" msgpack:"value"`
	}

	original := testStruct{Name: "test", Value: 42}

	serializers := []Serializer{
		NewJSONSerializer(),
		NewMsgpackSerializer(),
		NewGobSerializer(),
	}

	for _, serializer := range serializers {
		t.Run(serializer.ContentType(), func(t *testing.T) {
			// First serialize the data
			data, err := serializer.Serialize(original)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Test StringDeserializer interface
			stringDeser, ok := serializer.(StringDeserializer)
			if !ok {
				t.Fatalf("Serializer does not implement StringDeserializer")
			}

			// Test DeserializeString
			var result1 testStruct
			err = stringDeser.DeserializeString(string(data), &result1)
			if err != nil {
				t.Fatalf("DeserializeString failed: %v", err)
			}

			// Test regular Deserialize for comparison
			var result2 testStruct
			err = serializer.Deserialize(data, &result2)
			if err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			// Results should be identical
			if result1 != result2 {
				t.Errorf("DeserializeString result %+v != Deserialize result %+v", result1, result2)
			}

			if result1 != original {
				t.Errorf("DeserializeString result %+v != original %+v", result1, original)
			}
		})
	}
}

func TestStringDeserializerEdgeCases(t *testing.T) {
	serializers := []Serializer{
		NewJSONSerializer(),
		NewMsgpackSerializer(),
		NewGobSerializer(),
	}

	for _, serializer := range serializers {
		t.Run(serializer.ContentType(), func(t *testing.T) {
			stringDeser, ok := serializer.(StringDeserializer)
			if !ok {
				t.Fatalf("Serializer does not implement StringDeserializer")
			}

			// Test empty string
			var result string
			err := stringDeser.DeserializeString("", &result)
			if err == nil {
				t.Error("Expected error for empty string, got nil")
			}
		})
	}
}