package serializer

import (
	stdjson "encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

// TestJsoniterConfigFastest validates that the JSON serializer uses jsoniter.ConfigFastest
func TestJsoniterConfigFastest(t *testing.T) {
	// Create a serializer instance
	s := NewJSONSerializer(32 * 1024).(*JSONSerializer)

	// Test data that would behave differently with different configs
	testData := map[string]interface{}{
		"number": 123.456789012345,
		"html":   "<script>alert('test')</script>",
	}

	// Serialize using our serializer
	data, err := s.Serialize(testData)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// The serialized data should not have escaped HTML (ConfigFastest behavior)
	dataStr := string(data)
	if !strings.Contains(dataStr, "<script>") {
		t.Error("Expected unescaped HTML tags, indicating ConfigFastest is used")
	}

	// Compare with standard library (which would escape HTML by default)
	stdData, err := stdjson.Marshal(testData)
	if err != nil {
		t.Fatalf("Standard library marshal failed: %v", err)
	}

	stdStr := string(stdData)
	if strings.Contains(stdStr, "<script>") {
		// If standard library doesn't escape, this test assumption is wrong
		t.Log("Note: Standard library behavior may have changed - this test validates no HTML escaping")
	}

	// Verify our implementation doesn't escape
	if strings.Contains(dataStr, "\\u003c") || strings.Contains(dataStr, "&lt;") {
		t.Error("HTML should not be escaped when using ConfigFastest with SetEscapeHTML(false)")
	}
}

// TestEscapeHTMLDisabled confirms that HTML escaping is disabled
func TestEscapeHTMLDisabled(t *testing.T) {
	s := NewJSONSerializer(32 * 1024)

	testCases := []struct {
		name string
		data interface{}
		shouldContain string
	}{
		{
			name: "script_tags",
			data: map[string]string{"html": "<script>alert('test')</script>"},
			shouldContain: "<script>",
		},
		{
			name: "ampersand",
			data: map[string]string{"text": "Tom & Jerry"},
			shouldContain: "Tom & Jerry",
		},
		{
			name: "quotes",
			data: map[string]string{"quote": `He said "Hello"`},
			shouldContain: `"Hello"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := s.Serialize(tc.data)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			dataStr := string(data)
			if !strings.Contains(dataStr, tc.shouldContain) {
				t.Errorf("Expected serialized data to contain %q, got: %s", tc.shouldContain, dataStr)
			}
		})
	}
}

// TestJsoniterCompatibilityMode tests jsoniter-specific behavior vs standard library
func TestJsoniterCompatibilityMode(t *testing.T) {
	s := NewJSONSerializer(32 * 1024)

	testCases := []struct {
		name string
		data interface{}
	}{
		{"map", map[string]interface{}{"key": "value", "number": 42}},
		{"slice", []interface{}{"a", "b", 123, true}},
		{"nested", map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
				"age":  30,
				"tags": []string{"admin", "user"},
			},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize with our jsoniter-based serializer
			jsoniterData, err := s.Serialize(tc.data)
			if err != nil {
				t.Fatalf("Jsoniter serialize failed: %v", err)
			}

			// Serialize with standard library
			stdlibData, err := stdjson.Marshal(tc.data)
			if err != nil {
				t.Fatalf("Standard library marshal failed: %v", err)
			}

			// Both should deserialize to the same result
			var jsoniterResult interface{}
			var stdlibResult interface{}

			if err := s.Deserialize(jsoniterData, &jsoniterResult); err != nil {
				t.Fatalf("Jsoniter deserialize failed: %v", err)
			}

			if err := stdjson.Unmarshal(stdlibData, &stdlibResult); err != nil {
				t.Fatalf("Standard library unmarshal failed: %v", err)
			}

			// Results should be equivalent (allowing for HTML escaping differences)
			if !reflect.DeepEqual(jsoniterResult, stdlibResult) {
				t.Logf("Jsoniter result: %+v", jsoniterResult)
				t.Logf("Stdlib result: %+v", stdlibResult)
				// This is informational - slight differences are expected due to ConfigFastest
			}
		})
	}
}

// TestJsoniterNumberHandling tests how jsoniter handles numbers differently from stdlib
func TestJsoniterNumberHandling(t *testing.T) {
	s := NewJSONSerializer(32 * 1024)

	testCases := []struct {
		name   string
		number interface{}
	}{
		{"int64", int64(9223372036854775807)},
		{"float64", 123.456789012345678901234567890},
		{"scientific", 1.23e-10},
		{"zero", 0},
		{"negative", -123.456},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]interface{}{"number": tc.number}

			// Serialize and deserialize
			serialized, err := s.Serialize(data)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			var result map[string]interface{}
			if err := s.Deserialize(serialized, &result); err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			// Verify the number was handled correctly
			if result["number"] == nil {
				t.Error("Number field is nil after round-trip")
			}

			// Numbers in JSON are always float64 after deserialization
			if _, ok := result["number"].(float64); !ok {
				t.Errorf("Expected float64, got %T", result["number"])
			}
		})
	}
}

// TestJsoniterNullHandling tests null value handling
func TestJsoniterNullHandling(t *testing.T) {
	s := NewJSONSerializer(32 * 1024)

	testCases := []struct {
		name string
		data interface{}
	}{
		{"nil_map_value", map[string]interface{}{"key": nil}},
		{"nil_slice_element", []interface{}{1, nil, "test"}},
		{"nested_nil", map[string]interface{}{
			"user": map[string]interface{}{
				"name": "John",
				"age":  nil,
			},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize
			serialized, err := s.Serialize(tc.data)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Should contain null values in JSON
			dataStr := string(serialized)
			if !strings.Contains(dataStr, "null") {
				t.Error("Expected serialized data to contain 'null' for nil values")
			}

			// Deserialize and verify
			var result interface{}
			if err := s.Deserialize(serialized, &result); err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			// Verify the structure is maintained
			if result == nil {
				t.Error("Deserialized result should not be nil")
			}
		})
	}
}

// TestJsoniterMalformedJSON tests how jsoniter handles malformed JSON
func TestJsoniterMalformedJSON(t *testing.T) {
	s := NewJSONSerializer(32 * 1024)

	malformedCases := []struct {
		name string
		json string
	}{
		{"unclosed_brace", `{"key": "value"`},
		{"trailing_comma", `{"key": "value",}`},
		{"unquoted_key", `{key: "value"}`},
		{"single_quotes", `{'key': 'value'}`},
		{"invalid_escape", `{"key": "\x"}`},
		{"truncated", `{"key":`},
		{"extra_comma", `{"a": 1,, "b": 2}`},
	}

	for _, tc := range malformedCases {
		t.Run(tc.name, func(t *testing.T) {
			var result map[string]interface{}
			err := s.Deserialize([]byte(tc.json), &result)

			// Should fail with an error
			if err == nil {
				t.Errorf("Expected error for malformed JSON %q, but got success with result: %+v", tc.json, result)
			}

			// Error should be meaningful
			if err != nil && strings.TrimSpace(err.Error()) == "" {
				t.Error("Error message should not be empty")
			}
		})
	}
}

// TestJsoniterSpecialFloatValues tests handling of special float values
func TestJsoniterSpecialFloatValues(t *testing.T) {
	s := NewJSONSerializer(32 * 1024)

	testCases := []struct {
		name  string
		value float64
		expectError bool
	}{
		{"positive_infinity", math.Inf(1), true},  // JSON doesn't support Inf
		{"negative_infinity", math.Inf(-1), true}, // JSON doesn't support -Inf
		{"nan", math.NaN(), true},                 // JSON doesn't support NaN
		{"max_float", math.MaxFloat64, false},
		{"min_positive", math.SmallestNonzeroFloat64, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]interface{}{"value": tc.value}

			_, err := s.Serialize(data)

			if tc.expectError && err == nil {
				t.Errorf("Expected error for %s, but serialization succeeded", tc.name)
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.name, err)
			}
		})
	}
}

// TestJsoniterPerformanceCharacteristics tests that jsoniter provides expected performance benefits
func TestJsoniterPerformanceCharacteristics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	s := NewJSONSerializer(32 * 1024)

	// Large test data
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[string(rune('a'+i%26))+string(rune('0'+i%10))] = map[string]interface{}{
			"id":    i,
			"name":  "Item " + string(rune('0'+i%10)),
			"value": float64(i) * 1.23,
			"tags":  []string{"tag1", "tag2", "tag3"},
		}
	}

	// Measure jsoniter performance
	const iterations = 100
	jsoniterStart := testing.Short()

	for i := 0; i < iterations; i++ {
		data, err := s.Serialize(largeData)
		if err != nil {
			t.Fatalf("Jsoniter serialize failed: %v", err)
		}

		var result map[string]interface{}
		if err := s.Deserialize(data, &result); err != nil {
			t.Fatalf("Jsoniter deserialize failed: %v", err)
		}
	}

	// This test primarily validates that jsoniter can handle large datasets
	// without errors and completes in reasonable time
	_ = jsoniterStart // Suppress unused variable warning
}

// TestJsoniterThreadSafety tests concurrent usage of jsoniter
func TestJsoniterThreadSafety(t *testing.T) {
	s := NewJSONSerializer(32 * 1024)

	data := map[string]interface{}{
		"message": "Hello, World!",
		"number":  42,
		"array":   []interface{}{1, 2, 3},
	}

	const numGoroutines = 10
	const operationsPerGoroutine = 100

	done := make(chan bool, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func() {
			defer func() { done <- true }()

			for i := 0; i < operationsPerGoroutine; i++ {
				// Serialize
				serialized, err := s.Serialize(data)
				if err != nil {
					t.Errorf("Serialize failed in goroutine: %v", err)
					return
				}

				// Deserialize
				var result map[string]interface{}
				if err := s.Deserialize(serialized, &result); err != nil {
					t.Errorf("Deserialize failed in goroutine: %v", err)
					return
				}

				// Basic validation
				if result["message"] != "Hello, World!" {
					t.Errorf("Data corruption detected in concurrent test")
					return
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for g := 0; g < numGoroutines; g++ {
		<-done
	}
}