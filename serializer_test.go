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
			// Test nil value
			if s.name != "JSON" { // json.Marshal(nil) is valid
				_, err := s.serializer.Serialize(nil)
				if err == nil {
					t.Error("Expected error for nil value")
				}
			}

			// Test nil data
			var v testStruct
			err := s.serializer.Deserialize(nil, &v)
			if err == nil {
				t.Error("Expected error for nil data")
			}

			// Test invalid data
			err = s.serializer.Deserialize([]byte("invalid data"), &v)
			if err == nil {
				t.Error("Expected error for invalid data")
			}

			// Test nil reader
			err = s.serializer.DeserializeFrom(nil, &v)
			if err == nil {
				t.Error("Expected error for nil reader")
			} else if err.Error() != "reader is nil" {
				t.Errorf("Expected 'reader is nil' error, got: %v", err)
			}

			// Test nil writer
			err = s.serializer.SerializeTo(nil, &v)
			if err == nil {
				t.Error("Expected error for nil writer")
			} else if err.Error() != "writer is nil" {
				t.Errorf("Expected 'writer is nil' error, got: %v", err)
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
	registry.Register(serializer.JSON, serializer.NewJSONSerializer())
	registry.Register(serializer.Binary, serializer.NewGobSerializer())
	registry.Register(serializer.Msgpack, serializer.NewMsgpackSerializer())

	// Test getting registered serializers
	formats := map[string]serializer.Format{
		"JSON":    serializer.JSON,
		"Gob":     serializer.Binary,
		"MsgPack": serializer.Msgpack,
	}

	for name, format := range formats {
		t.Run(name, func(t *testing.T) {
			ser, ok := registry.Get(format)
			if !ok {
				t.Errorf("Serializer %s not found in registry", name)
			}
			if ser == nil {
				t.Errorf("Got nil serializer for %s", name)
			}
		})
	}

	// Test getting non-existent serializer
	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Expected false for non-existent serializer")
	}

	// Test New factory method
	for name, format := range formats {
		t.Run(name, func(t *testing.T) {
			ser, err := registry.New(format)
			if err != nil {
				t.Errorf("Failed to create serializer %s: %v", name, err)
			}
			if ser == nil {
				t.Errorf("Got nil serializer for %s", name)
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

// TestUniformSerialization verifies that all serializers produce consistent results
func TestUniformSerialization(t *testing.T) {
	// Test data with various types
	testData := testStruct{
		String:    "test",
		Int:       42,
		Float:     3.14,
		Bool:      true,
		Time:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Slice:     []string{"a", "b", "c"},
		Map:       map[string]int{"x": 1, "y": 2},
		Interface: "interface value",
	}

	// Test each serializer
	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			// Serialize the data
			bytes, err := s.serializer.Serialize(testData)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Get the type information
			valueType, err := s.serializer.GetType(bytes)
			if err != nil {
				t.Fatalf("GetType failed: %v", err)
			}
			if valueType != serializer.TypeStruct {
				t.Errorf("Expected type %s, got %s", serializer.TypeStruct, valueType)
			}

			// Deserialize into a new struct
			var result testStruct
			err = s.serializer.Deserialize(bytes, &result)
			if err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			// Verify the result matches the input
			if result.String != testData.String {
				t.Errorf("String mismatch: expected %q, got %q", testData.String, result.String)
			}
			if result.Int != testData.Int {
				t.Errorf("Int mismatch: expected %d, got %d", testData.Int, result.Int)
			}
			if result.Float != testData.Float {
				t.Errorf("Float mismatch: expected %f, got %f", testData.Float, result.Float)
			}
			if result.Bool != testData.Bool {
				t.Errorf("Bool mismatch: expected %v, got %v", testData.Bool, result.Bool)
			}
			if !result.Time.Equal(testData.Time) {
				t.Errorf("Time mismatch: expected %v, got %v", testData.Time, result.Time)
			}
			if !reflect.DeepEqual(result.Slice, testData.Slice) {
				t.Errorf("Slice mismatch: expected %v, got %v", testData.Slice, result.Slice)
			}
			if !reflect.DeepEqual(result.Map, testData.Map) {
				t.Errorf("Map mismatch: expected %v, got %v", testData.Map, result.Map)
			}
			if result.Interface != testData.Interface {
				t.Errorf("Interface mismatch: expected %v, got %v", testData.Interface, result.Interface)
			}
		})
	}
}

// TestCrossSerializerCompatibility verifies that data serialized with one serializer
// can be deserialized with another
func TestCrossSerializerCompatibility(t *testing.T) {
	// Test data
	testData := map[string]any{
		"string": "test",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"slice":  []string{"a", "b", "c"},
		"map":    map[string]int{"x": 1, "y": 2},
	}

	// Test each serializer combination
	for _, s1 := range testSerializers {
		for _, s2 := range testSerializers {
			t.Run(fmt.Sprintf("%s_to_%s", s1.name, s2.name), func(t *testing.T) {
				// Serialize with first serializer
				bytes, err := s1.serializer.Serialize(testData)
				if err != nil {
					t.Fatalf("Serialize with %s failed: %v", s1.name, err)
				}

				// Deserialize with second serializer
				var result map[string]any
				err = s2.serializer.Deserialize(bytes, &result)
				if err != nil {
					t.Fatalf("Deserialize with %s failed: %v", s2.name, err)
				}

				// Verify the result matches the input
				if !reflect.DeepEqual(result, testData) {
					t.Errorf("Data mismatch: expected %v, got %v", testData, result)
				}
			})
		}
	}
}

// TestTypePreservation verifies that type information is preserved during serialization
func TestTypePreservation(t *testing.T) {
	// Test cases with different types
	testCases := []struct {
		name  string
		value any
		typ   serializer.Type
	}{
		{"string", "test", serializer.TypeString},
		{"int", 42, serializer.TypeInt},
		{"float", 3.14, serializer.TypeFloat},
		{"bool", true, serializer.TypeBool},
		{"slice", []string{"a", "b"}, serializer.TypeSlice},
		{"map", map[string]int{"x": 1}, serializer.TypeMap},
		{"struct", struct{ X int }{42}, serializer.TypeStruct},
	}

	// Test each serializer
	for _, s := range testSerializers {
		t.Run(s.name, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					// Serialize the value
					bytes, err := s.serializer.Serialize(tc.value)
					if err != nil {
						t.Fatalf("Serialize failed: %v", err)
					}

					// Get the type information
					valueType, err := s.serializer.GetType(bytes)
					if err != nil {
						t.Fatalf("GetType failed: %v", err)
					}
					if valueType != tc.typ {
						t.Errorf("Expected type %s, got %s", tc.typ, valueType)
					}

					// Deserialize and verify type
					var result any
					err = s.serializer.Deserialize(bytes, &result)
					if err != nil {
						t.Fatalf("Deserialize failed: %v", err)
					}

					// Verify the type of the result
					resultType := getType(result)
					if resultType != tc.typ {
						t.Errorf("Result type mismatch: expected %s, got %s", tc.typ, resultType)
					}
				})
			}
		})
	}
}

// getType is a helper function to determine the type of a value
func getType(v any) serializer.Type {
	if v == nil {
		return serializer.TypeNil
	}

	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return serializer.TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return serializer.TypeInt
	case reflect.Float32, reflect.Float64:
		return serializer.TypeFloat
	case reflect.Bool:
		return serializer.TypeBool
	case reflect.Slice:
		return serializer.TypeSlice
	case reflect.Map:
		return serializer.TypeMap
	case reflect.Struct:
		return serializer.TypeStruct
	default:
		return serializer.TypeNil
	}
}
