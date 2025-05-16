package serializer_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/MichaelAJay/go-serializer"
)

// Type represents the type of a serialized value
type Type string

const (
	TypeNil    Type = "nil"
	TypeString Type = "string"
	TypeInt    Type = "int"
	TypeFloat  Type = "float"
	TypeBool   Type = "bool"
	TypeSlice  Type = "slice"
	TypeMap    Type = "map"
	TypeStruct Type = "struct"
)

// testStruct is a struct with various field types for testing
type testStruct struct {
	String    string
	Int       int
	Float     float64
	Bool      bool
	Time      time.Time
	Slice     []string
	Map       map[string]int
	Ptr       *string
	Interface any
}

// testCases contains various test cases for serialization
var testCases = []struct {
	name     string
	value    any
	expected any
	jsonType any // Expected type after JSON deserialization
}{
	{
		name:     "string",
		value:    "hello world",
		expected: "hello world",
		jsonType: "hello world",
	},
	{
		name:     "int",
		value:    42,
		expected: 42,
		jsonType: float64(42), // JSON numbers are float64
	},
	{
		name:     "float",
		value:    3.14,
		expected: 3.14,
		jsonType: 3.14,
	},
	{
		name:     "bool",
		value:    true,
		expected: true,
		jsonType: true,
	},
	{
		name:     "slice",
		value:    []string{"a", "b", "c"},
		expected: []string{"a", "b", "c"},
		jsonType: []any{"a", "b", "c"}, // JSON arrays are []any
	},
	{
		name:     "map",
		value:    map[string]int{"a": 1, "b": 2},
		expected: map[string]int{"a": 1, "b": 2},
		jsonType: map[string]any{"a": float64(1), "b": float64(2)}, // JSON numbers are float64
	},
	{
		name: "struct",
		value: testStruct{
			String:    "test",
			Int:       42,
			Float:     3.14,
			Bool:      true,
			Time:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Slice:     []string{"a", "b"},
			Map:       map[string]int{"x": 1},
			Ptr:       nil,
			Interface: "interface value",
		},
		expected: testStruct{
			String:    "test",
			Int:       42,
			Float:     3.14,
			Bool:      true,
			Time:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Slice:     []string{"a", "b"},
			Map:       map[string]int{"x": 1},
			Ptr:       nil,
			Interface: "interface value",
		},
		jsonType: map[string]any{
			"String":    "test",
			"Int":       float64(42),
			"Float":     3.14,
			"Bool":      true,
			"Time":      "2024-01-01T00:00:00Z",
			"Slice":     []any{"a", "b"},
			"Map":       map[string]any{"x": float64(1)},
			"Ptr":       nil,
			"Interface": "interface value",
		},
	},
}

// testSerializers contains all serializer implementations to test
var testSerializers = []struct {
	name       string
	serializer serializer.Serializer
}{
	{"JSON", serializer.NewJSONSerializer()},
	{"Gob", serializer.NewGobSerializer()},
	{"MsgPack", serializer.NewMsgpackSerializer()},
}

func TestSerialization(t *testing.T) {
	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					data, err := s.serializer.Serialize(tc.value)
					if err != nil {
						t.Fatalf("Serialize failed: %v", err)
					}

					var got any
					switch s.name {
					case "JSON":
						got = tc.jsonType
						if err := s.serializer.Deserialize(data, &got); err != nil {
							t.Fatalf("Deserialize failed: %v", err)
						}
						if !compareValues(tc.expected, got) {
							t.Errorf("Expected %v, got %v", tc.expected, got)
						}
					case "Gob", "MsgPack":
						// Use pointer to concrete type for deserialization
						var ptr any
						switch tc.value.(type) {
						case string:
							var v string
							ptr = &v
						case int:
							var v int
							ptr = &v
						case float64:
							var v float64
							ptr = &v
						case bool:
							var v bool
							ptr = &v
						case []string:
							var v []string
							ptr = &v
						case map[string]int:
							var v map[string]int
							ptr = &v
						case testStruct:
							var v testStruct
							ptr = &v
						}
						if err := s.serializer.Deserialize(data, ptr); err != nil {
							t.Fatalf("Deserialize failed: %v", err)
						}
						// Dereference for comparison
						got = deref(ptr)
						if !compareValues(tc.expected, got) {
							t.Errorf("Expected %v, got %v", tc.expected, got)
						}
					}
				})
			}
		})
	}
}

func TestStreaming(t *testing.T) {
	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					var buf bytes.Buffer
					if err := s.serializer.SerializeTo(&buf, tc.value); err != nil {
						t.Fatalf("SerializeTo failed: %v", err)
					}

					switch s.name {
					case "JSON":
						got := tc.jsonType
						if err := s.serializer.DeserializeFrom(&buf, &got); err != nil {
							t.Fatalf("DeserializeFrom failed: %v", err)
						}
						if !compareValues(tc.expected, got) {
							t.Errorf("Expected %v, got %v", tc.expected, got)
						}
					case "Gob", "MsgPack":
						var ptr any
						switch tc.value.(type) {
						case string:
							var v string
							ptr = &v
						case int:
							var v int
							ptr = &v
						case float64:
							var v float64
							ptr = &v
						case bool:
							var v bool
							ptr = &v
						case []string:
							var v []string
							ptr = &v
						case map[string]int:
							var v map[string]int
							ptr = &v
						case testStruct:
							var v testStruct
							ptr = &v
						}
						if err := s.serializer.DeserializeFrom(&buf, ptr); err != nil {
							t.Fatalf("DeserializeFrom failed: %v", err)
						}
						got := deref(ptr)
						if !compareValues(tc.expected, got) {
							t.Errorf("Expected %v, got %v", tc.expected, got)
						}
					}
				})
			}
		})
	}
}

func TestErrorCases(t *testing.T) {
	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			// Test deserializing into nil
			err := s.serializer.Deserialize([]byte("{}"), nil)
			if err == nil {
				t.Error("Expected error when deserializing into nil")
			}

			// Test deserializing into non-pointer
			var v string
			err = s.serializer.Deserialize([]byte("{}"), v)
			if err == nil {
				t.Error("Expected error when deserializing into non-pointer")
			}

			// Test deserializing invalid data
			err = s.serializer.Deserialize([]byte("invalid"), &v)
			if err == nil {
				t.Error("Expected error when deserializing invalid data")
			}
		})
	}
}

func TestContentType(t *testing.T) {
	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			contentType := s.serializer.ContentType()
			switch s.name {
			case "JSON":
				if contentType != "application/json" {
					t.Errorf("Expected content type application/json, got %s", contentType)
				}
			case "Gob":
				if contentType != "application/x-gob" {
					t.Errorf("Expected content type application/x-gob, got %s", contentType)
				}
			case "MsgPack":
				if contentType != "application/x-msgpack" {
					t.Errorf("Expected content type application/x-msgpack, got %s", contentType)
				}
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := serializer.NewRegistry()

	// Test registering serializers
	for _, s := range testSerializers {
		registry.Register(serializer.Format(s.name), s.serializer)
	}

	// Test getting serializers
	for _, s := range testSerializers {
		got, ok := registry.Get(serializer.Format(s.name))
		if !ok {
			t.Errorf("Serializer %s not found in registry", s.name)
		}
		if got != s.serializer {
			t.Errorf("Got different serializer for %s", s.name)
		}
	}

	// Test getting non-existent serializer
	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Expected false when getting non-existent serializer")
	}

	// Test creating new serializer
	for _, s := range testSerializers {
		got, err := registry.New(serializer.Format(s.name))
		if err != nil {
			t.Errorf("Error creating serializer %s: %v", s.name, err)
		}
		if got != s.serializer {
			t.Errorf("Got different serializer for %s", s.name)
		}
	}

	// Test creating non-existent serializer
	_, err := registry.New("nonexistent")
	if err == nil {
		t.Error("Expected error when creating non-existent serializer")
	}
}

// Helper functions for comparing values
func compareValues(expected, got any) bool {
	if expected == nil && got == nil {
		return true
	}
	if expected == nil || got == nil {
		return false
	}

	switch exp := expected.(type) {
	case string, bool:
		return exp == got
	case int:
		// Handle JSON's conversion of integers to float64
		if gf, ok := got.(float64); ok {
			return float64(exp) == gf
		}
		return exp == got
	case float64:
		return exp == got
	case []string:
		if got, ok := got.([]string); ok {
			return compareSlices(exp, got)
		}
		if got, ok := got.([]any); ok {
			return compareInterfaceSlices(exp, got)
		}
		return false
	case map[string]int:
		if got, ok := got.(map[string]int); ok {
			return compareMaps(exp, got)
		}
		if got, ok := got.(map[string]any); ok {
			return compareInterfaceMap(exp, got)
		}
		return false
	case testStruct:
		if got, ok := got.(testStruct); ok {
			return compareStructs(exp, got)
		}
		if got, ok := got.(map[string]any); ok {
			return compareStructWithMap(exp, got)
		}
		return false
	default:
		return reflect.DeepEqual(expected, got)
	}
}

func compareStructs(a, b testStruct) bool {
	return a.String == b.String &&
		a.Int == b.Int &&
		a.Float == b.Float &&
		a.Bool == b.Bool &&
		a.Time.Equal(b.Time) &&
		compareSlices(a.Slice, b.Slice) &&
		compareMaps(a.Map, b.Map) &&
		((a.Ptr == nil && b.Ptr == nil) || (a.Ptr != nil && b.Ptr != nil && *a.Ptr == *b.Ptr)) &&
		a.Interface == b.Interface
}

func compareStructWithMap(s testStruct, m map[string]any) bool {
	if m["String"] != s.String {
		return false
	}
	if m["Int"] != float64(s.Int) { // JSON numbers are float64
		return false
	}
	if m["Float"] != s.Float {
		return false
	}
	if m["Bool"] != s.Bool {
		return false
	}
	if m["Time"] != s.Time.Format(time.RFC3339) {
		return false
	}
	if !compareInterfaceSlices(s.Slice, m["Slice"].([]any)) {
		return false
	}
	if !compareInterfaceMap(s.Map, m["Map"].(map[string]any)) {
		return false
	}
	if m["Ptr"] != nil {
		return false
	}
	if m["Interface"] != s.Interface {
		return false
	}
	return true
}

func compareInterfaceSlices(a []string, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func compareInterfaceMap(a map[string]int, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != float64(v) { // JSON numbers are float64
			return false
		}
	}
	return true
}

func compareSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func compareMaps(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func deref(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		return rv.Elem().Interface()
	}
	return v
}

func TestUniformSerialization(t *testing.T) {
	// Test that serialization is format-specific and cross-format deserialization fails
	// This test validates that each serializer properly rejects data it can't understand

	// Test data
	data := map[string]any{
		"name":  "Test",
		"value": 123,
		"tags":  []string{"a", "b", "c"},
	}

	// Serialize with each format
	serialized := make(map[string][]byte)
	for _, s := range testSerializers {
		bytes, err := s.serializer.Serialize(data)
		if err != nil {
			t.Fatalf("Serialize failed for %s: %v", s.name, err)
		}
		serialized[s.name] = bytes
	}

	// Test cross-format deserialization - should fail for different formats
	for _, source := range testSerializers {
		for _, target := range testSerializers {
			t.Run(fmt.Sprintf("%s_to_%s", source.name, target.name), func(t *testing.T) {
				var result map[string]any
				err := target.serializer.Deserialize(serialized[source.name], &result)

				if source.name == target.name {
					// Same format - should succeed
					if err != nil {
						t.Errorf("Expected success for same format, got error: %v", err)
					}
					// Validate deserialized data
					if fmt.Sprintf("%v", result["name"]) != fmt.Sprintf("%v", data["name"]) {
						t.Errorf("Expected name %v, got %v", data["name"], result["name"])
					}
				} else {
					// Different formats - should fail with error
					if err == nil {
						t.Errorf("Expected error when deserializing from %s to %s, got success",
							source.name, target.name)
					}
					// Error message content is not standardized, so we just check that it exists
				}
			})
		}
	}
}

func TestCrossSerializerCompatibility(t *testing.T) {
	// Test that cross-format deserialization properly fails for complex types

	// Create a complex object
	original := testStruct{
		String:    "test",
		Int:       42,
		Float:     3.14,
		Bool:      true,
		Time:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Slice:     []string{"a", "b"},
		Map:       map[string]int{"x": 1},
		Ptr:       nil,
		Interface: "interface value",
	}

	// Serialize with each format
	serialized := make(map[string][]byte)
	for _, s := range testSerializers {
		bytes, err := s.serializer.Serialize(original)
		if err != nil {
			t.Fatalf("Serialize failed for %s: %v", s.name, err)
		}
		serialized[s.name] = bytes
	}

	// Test cross-format deserialization - should fail for different formats
	for _, source := range testSerializers {
		for _, target := range testSerializers {
			t.Run(fmt.Sprintf("%s_to_%s", source.name, target.name), func(t *testing.T) {
				var result testStruct
				err := target.serializer.Deserialize(serialized[source.name], &result)

				if source.name == target.name {
					// Same format - should succeed
					if err != nil {
						t.Errorf("Expected success for same format, got error: %v", err)
					}
					// Validate deserialized data
					if !compareStructs(original, result) {
						t.Errorf("Deserialized struct does not match original")
					}
				} else {
					// Different formats - should fail with error
					if err == nil {
						t.Errorf("Expected error when deserializing from %s to %s, got success",
							source.name, target.name)
					}
					// Error message content is not standardized, so we just check that it exists
				}
			})
		}
	}
}

func TestVersion(t *testing.T) {
	// Test VersionString
	version := serializer.VersionString()
	if version != serializer.Version {
		t.Errorf("Expected version %q, got %q", serializer.Version, version)
	}

	// Test VersionInfo
	info := serializer.VersionInfo()
	if info["major"] != serializer.VersionMajor {
		t.Errorf("Expected major version %d, got %d", serializer.VersionMajor, info["major"])
	}
	if info["minor"] != serializer.VersionMinor {
		t.Errorf("Expected minor version %d, got %d", serializer.VersionMinor, info["minor"])
	}
	if info["patch"] != serializer.VersionPatch {
		t.Errorf("Expected patch version %d, got %d", serializer.VersionPatch, info["patch"])
	}
}
