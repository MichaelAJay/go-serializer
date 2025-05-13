package serializer_test

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/MichaelAJay/go-serializer"
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
			Interface: "interface value",
		},
		jsonType: map[string]any{
			"String":    "test",
			"Int":       float64(42),
			"Float":     3.14,
			"Bool":      true,
			"Time":      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			"Slice":     []any{"a", "b"},
			"Map":       map[string]any{"x": float64(1)},
			"Interface": "interface value",
		},
	},
}

// testSerializers contains all serializer implementations to test
var testSerializers = []struct {
	name       string
	serializer serializer.Serializer
}{
	{"JSON", &serializer.JSONSerializer{}},
	{"Gob", &serializer.GobSerializer{}},
	{"MsgPack", &serializer.MsgPackSerializer{}},
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
			// Test nil value
			if s.name != "JSON" { // json.Marshal(nil) is valid
				_, err := s.serializer.Serialize(nil)
				if err == nil {
					t.Error("Expected error for nil value")
				}
			}

			// Test invalid data
			var v testStruct
			err := s.serializer.Deserialize([]byte("invalid data"), &v)
			if err == nil {
				t.Error("Expected error for invalid data")
			}

			// Test nil reader/writer (skip for JSON, which panics)
			if s.name != "JSON" {
				err = s.serializer.DeserializeFrom(nil, &v)
				if err == nil {
					t.Error("Expected error for nil reader")
				}
				err = s.serializer.SerializeTo(nil, &v)
				if err == nil {
					t.Error("Expected error for nil writer")
				}
			}
		})
	}
}

func TestContentType(t *testing.T) {
	expectedTypes := map[string]string{
		"JSON":    "application/json",
		"Gob":     "application/octet-stream",
		"MsgPack": "application/msgpack",
	}

	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			contentType := s.serializer.ContentType()
			expected := expectedTypes[s.name]
			if contentType != expected {
				t.Errorf("Expected content type %s, got %s", expected, contentType)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	registry := serializer.NewRegistry()

	// Register serializers
	registry.Register(serializer.JSON, &serializer.JSONSerializer{})
	registry.Register(serializer.Binary, &serializer.GobSerializer{})
	registry.Register(serializer.Msgpack, &serializer.MsgPackSerializer{})

	// Test getting registered serializers
	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			ser, ok := registry.Get(serializer.Format(s.name))
			if !ok {
				t.Errorf("Serializer %s not found in registry", s.name)
			}
			if ser == nil {
				t.Errorf("Got nil serializer for %s", s.name)
			}
		})
	}

	// Test getting non-existent serializer
	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Expected false for non-existent serializer")
	}

	// Test New factory method
	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			ser, err := registry.New(serializer.Format(s.name))
			if err != nil {
				t.Errorf("Failed to create serializer %s: %v", s.name, err)
			}
			if ser == nil {
				t.Errorf("Got nil serializer for %s", s.name)
			}
		})
	}

	// Test New with non-existent format
	_, err := registry.New("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent format")
	}
}

// compareValues compares two values for equality
func compareValues(expected, got any) bool {
	switch e := expected.(type) {
	case testStruct:
		g, ok := got.(testStruct)
		if !ok {
			// Try to compare with map for JSON case
			if m, ok := got.(map[string]any); ok {
				return compareStructWithMap(e, m)
			}
			return false
		}
		return e.String == g.String &&
			e.Int == g.Int &&
			e.Float == g.Float &&
			e.Bool == g.Bool &&
			e.Time.Equal(g.Time) &&
			compareSlices(e.Slice, g.Slice) &&
			compareMaps(e.Map, g.Map) &&
			e.Interface == g.Interface
	case []string:
		// Handle both []string and []any cases
		switch g := got.(type) {
		case []string:
			return compareSlices(e, g)
		case []any:
			return compareInterfaceSlices(e, g)
		default:
			return false
		}
	case map[string]int:
		// Handle both map[string]int and map[string]any cases
		switch g := got.(type) {
		case map[string]int:
			return compareMaps(e, g)
		case map[string]any:
			return compareInterfaceMap(e, g)
		default:
			return false
		}
	default:
		// Handle numeric type conversions
		switch e := e.(type) {
		case int:
			switch g := got.(type) {
			case int:
				return e == g
			case float64:
				return float64(e) == g
			default:
				return false
			}
		default:
			return expected == got
		}
	}
}

// compareStructWithMap compares a testStruct with a map[string]any
func compareStructWithMap(s testStruct, m map[string]any) bool {
	// Compare each field
	if s.String != m["String"] {
		return false
	}
	if float64(s.Int) != m["Int"].(float64) {
		return false
	}
	if s.Float != m["Float"].(float64) {
		return false
	}
	if s.Bool != m["Bool"].(bool) {
		return false
	}

	// Handle time.Time serialization in JSON
	timeStr, ok := m["Time"].(string)
	if !ok {
		return false
	}
	jsonTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return false
	}
	if !s.Time.Equal(jsonTime) {
		return false
	}

	if !compareInterfaceSlices(s.Slice, m["Slice"].([]any)) {
		return false
	}
	if !compareInterfaceMap(s.Map, m["Map"].(map[string]any)) {
		return false
	}
	if s.Interface != m["Interface"] {
		return false
	}
	return true
}

// compareInterfaceSlices compares a []string with a []any
func compareInterfaceSlices(a []string, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i].(string) {
			return false
		}
	}
	return true
}

// compareInterfaceMap compares a map[string]int with a map[string]any
func compareInterfaceMap(a map[string]int, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if float64(v) != b[k].(float64) {
			return false
		}
	}
	return true
}

// compareSlices compares two string slices
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

// compareMaps compares two string-int maps
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

// deref returns the value pointed to by a pointer, or the value itself if not a pointer
func deref(v any) any {
	// Use reflection to dereference
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		return val.Elem().Interface()
	}
	return v
}
