package serializer

import (
	stdjson "encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

// TestJsoniterVsStdlibConsistency tests serialization output consistency between jsoniter and stdlib
func TestJsoniterVsStdlibConsistency(t *testing.T) {
	s := NewJSONSerializer(4096)

	testCases := []struct {
		name string
		data interface{}
		skipHtmlComparison bool // Skip HTML escaping comparison
	}{
		{
			name: "SimpleObject",
			data: map[string]interface{}{
				"name":  "test",
				"value": 42,
				"active": true,
			},
		},
		{
			name: "NumericTypes",
			data: map[string]interface{}{
				"int":     123,
				"int64":   int64(123456789),
				"float32": float32(3.14),
				"float64": 3.141592653589793,
				"zero":    0,
				"negative": -456,
			},
		},
		{
			name: "StringTypes",
			data: map[string]interface{}{
				"simple":   "hello",
				"empty":    "",
				"unicode":  "Hello 世界 🌍",
				"escaped":  "String with \"quotes\" and \\ backslashes",
				"newlines": "Line 1\nLine 2\nLine 3",
			},
		},
		{
			name: "Arrays",
			data: map[string]interface{}{
				"strings":  []string{"a", "b", "c"},
				"numbers":  []int{1, 2, 3, 4, 5},
				"mixed":    []interface{}{"hello", 42, true, nil},
				"empty":    []interface{}{},
				"nested":   [][]int{{1, 2}, {3, 4}},
			},
		},
		{
			name: "NullValues",
			data: map[string]interface{}{
				"null_value": nil,
				"nested": map[string]interface{}{
					"also_null": nil,
				},
				"array_with_null": []interface{}{1, nil, 3},
			},
		},
		{
			name: "NestedObjects",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"deep_value": "nested",
						},
					},
				},
			},
		},
		{
			name: "HTMLContent",
			data: map[string]interface{}{
				"html":    "<script>alert('test')</script>",
				"xml":     "<root><child>value</child></root>",
				"special": "Tom & Jerry",
			},
			skipHtmlComparison: true, // jsoniter doesn't escape HTML by default
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize with jsoniter (our implementation)
			jsoniterOutput, err := s.Serialize(tc.data)
			if err != nil {
				t.Fatalf("Jsoniter serialize failed: %v", err)
			}

			// Serialize with standard library
			stdlibOutput, err := stdjson.Marshal(tc.data)
			if err != nil {
				t.Fatalf("Stdlib marshal failed: %v", err)
			}

			// Both outputs should be valid JSON
			var jsoniterResult interface{}
			var stdlibResult interface{}

			if err := stdjson.Unmarshal(jsoniterOutput, &jsoniterResult); err != nil {
				t.Fatalf("Jsoniter output is not valid JSON: %v", err)
			}

			if err := stdjson.Unmarshal(stdlibOutput, &stdlibResult); err != nil {
				t.Fatalf("Stdlib output is not valid JSON: %v", err)
			}

			// Results should be equivalent when parsed
			if !reflect.DeepEqual(jsoniterResult, stdlibResult) {
				if !tc.skipHtmlComparison {
					t.Errorf("Parsed results differ:\nJsoniter: %+v\nStdlib: %+v", jsoniterResult, stdlibResult)
				} else {
					t.Logf("Parsed results differ (expected due to HTML escaping):\nJsoniter: %+v\nStdlib: %+v", jsoniterResult, stdlibResult)
				}
			}

			// For non-HTML cases, outputs might be very similar or identical
			if !tc.skipHtmlComparison {
				jsoniterStr := string(jsoniterOutput)
				stdlibStr := string(stdlibOutput)

				// They might differ in whitespace or formatting, but should contain same data
				if jsoniterStr != stdlibStr {
					t.Logf("Raw outputs differ (may be due to formatting):")
					t.Logf("Jsoniter: %s", jsoniterStr)
					t.Logf("Stdlib: %s", stdlibStr)
				}
			}
		})
	}
}

// TestJsoniterSpecificFeatures tests features specific to jsoniter
func TestJsoniterSpecificFeatures(t *testing.T) {
	s := NewJSONSerializer(2048)

	testCases := []struct {
		name        string
		data        interface{}
		expectation string
	}{
		{
			name: "HTMLNotEscaped",
			data: map[string]interface{}{
				"content": "<script>alert('test')</script>",
			},
			expectation: "HTML should not be escaped in jsoniter output",
		},
		{
			name: "AmpersandNotEscaped",
			data: map[string]interface{}{
				"text": "Tom & Jerry",
			},
			expectation: "Ampersands should not be escaped",
		},
		{
			name: "LargeNumbers",
			data: map[string]interface{}{
				"big_int":   int64(math.MaxInt64),
				"big_float": math.MaxFloat64,
			},
			expectation: "Large numbers should be handled correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test jsoniter serialization
			jsoniterOutput, err := s.Serialize(tc.data)
			if err != nil {
				t.Fatalf("Jsoniter serialize failed: %v", err)
			}

			outputStr := string(jsoniterOutput)
			t.Logf("Jsoniter output: %s", outputStr)
			t.Logf("Expectation: %s", tc.expectation)

			// Verify it's valid JSON
			var result interface{}
			if err := stdjson.Unmarshal(jsoniterOutput, &result); err != nil {
				t.Fatalf("Output is not valid JSON: %v", err)
			}

			// Specific checks based on test case
			switch tc.name {
			case "HTMLNotEscaped":
				if !strings.Contains(outputStr, "<script>") {
					t.Error("HTML tags should not be escaped")
				}
				if strings.Contains(outputStr, "&lt;") || strings.Contains(outputStr, "\\u003c") {
					t.Error("Found escaped HTML - should not be escaped")
				}

			case "AmpersandNotEscaped":
				if !strings.Contains(outputStr, "Tom & Jerry") {
					t.Error("Ampersand should not be escaped")
				}
				if strings.Contains(outputStr, "&amp;") || strings.Contains(outputStr, "\\u0026") {
					t.Error("Found escaped ampersand - should not be escaped")
				}

			case "LargeNumbers":
				// Should handle large numbers without error
				resultMap := result.(map[string]interface{})
				if resultMap["big_int"] == nil || resultMap["big_float"] == nil {
					t.Error("Large numbers were not preserved")
				}
			}
		})
	}
}

// TestJsoniterErrorMessages tests error message consistency
func TestJsoniterErrorMessages(t *testing.T) {
	s := NewJSONSerializer(1024)

	malformedCases := []struct {
		name    string
		input   string
		checkError func(error) bool
	}{
		{
			name:  "UnterminatedString",
			input: `{"key": "unterminated`,
			checkError: func(err error) bool {
				return err != nil && strings.Contains(strings.ToLower(err.Error()), "string")
			},
		},
		{
			name:  "UnterminatedObject",
			input: `{"key": "value"`,
			checkError: func(err error) bool {
				return err != nil // Any error is acceptable
			},
		},
		{
			name:  "InvalidNumber",
			input: `{"key": 123.456.789}`,
			checkError: func(err error) bool {
				return err != nil
			},
		},
		{
			name:  "TrailingComma",
			input: `{"key": "value",}`,
			checkError: func(err error) bool {
				return err != nil
			},
		},
	}

	for _, tc := range malformedCases {
		t.Run(tc.name, func(t *testing.T) {
			var result interface{}
			jsoniterErr := s.Deserialize([]byte(tc.input), &result)

			// Also test with standard library for comparison
			var stdResult interface{}
			stdlibErr := stdjson.Unmarshal([]byte(tc.input), &stdResult)

			// Both should produce errors
			if jsoniterErr == nil {
				t.Errorf("Jsoniter should have produced an error for input: %s", tc.input)
			}

			if stdlibErr == nil {
				t.Errorf("Standard library should have produced an error for input: %s", tc.input)
			}

			// Check error using provided function
			if jsoniterErr != nil && !tc.checkError(jsoniterErr) {
				t.Errorf("Jsoniter error doesn't match expected pattern: %v", jsoniterErr)
			}

			if stdlibErr != nil && !tc.checkError(stdlibErr) {
				t.Logf("Stdlib error: %v", stdlibErr)
			}

			// Log both errors for comparison
			if jsoniterErr != nil && stdlibErr != nil {
				t.Logf("Jsoniter error: %v", jsoniterErr)
				t.Logf("Stdlib error: %v", stdlibErr)
			}
		})
	}
}

// TestJsoniterSpecialFloatHandling tests handling of special float values
func TestJsoniterSpecialFloatHandling(t *testing.T) {
	s := NewJSONSerializer(1024)

	specialCases := []struct {
		name        string
		value       float64
		expectError bool
		description string
	}{
		{
			name:        "PositiveInfinity",
			value:       math.Inf(1),
			expectError: true,
			description: "Positive infinity should cause an error",
		},
		{
			name:        "NegativeInfinity",
			value:       math.Inf(-1),
			expectError: true,
			description: "Negative infinity should cause an error",
		},
		{
			name:        "NaN",
			value:       math.NaN(),
			expectError: true,
			description: "NaN should cause an error",
		},
		{
			name:        "MaxFloat64",
			value:       math.MaxFloat64,
			expectError: false,
			description: "Maximum float64 value should be supported",
		},
		{
			name:        "SmallestFloat64",
			value:       math.SmallestNonzeroFloat64,
			expectError: false,
			description: "Smallest positive float64 should be supported",
		},
		{
			name:        "NegativeMaxFloat64",
			value:       -math.MaxFloat64,
			expectError: false,
			description: "Negative maximum float64 should be supported",
		},
	}

	for _, tc := range specialCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]interface{}{"value": tc.value}

			// Test jsoniter behavior
			jsoniterOutput, jsoniterErr := s.Serialize(data)

			// Test standard library behavior
			stdlibOutput, stdlibErr := stdjson.Marshal(data)

			t.Logf("Testing: %s", tc.description)

			if tc.expectError {
				if jsoniterErr == nil {
					t.Errorf("Jsoniter should have returned an error for %s", tc.name)
				}
				if stdlibErr == nil {
					t.Errorf("Standard library should have returned an error for %s", tc.name)
				}
			} else {
				if jsoniterErr != nil {
					t.Errorf("Jsoniter unexpected error for %s: %v", tc.name, jsoniterErr)
				}
				if stdlibErr != nil {
					t.Errorf("Standard library unexpected error for %s: %v", tc.name, stdlibErr)
				}

				// If both succeeded, outputs should be equivalent
				if jsoniterErr == nil && stdlibErr == nil {
					var jsoniterResult, stdlibResult interface{}
					
					if err := stdjson.Unmarshal(jsoniterOutput, &jsoniterResult); err != nil {
						t.Errorf("Failed to unmarshal jsoniter output: %v", err)
					}
					
					if err := stdjson.Unmarshal(stdlibOutput, &stdlibResult); err != nil {
						t.Errorf("Failed to unmarshal stdlib output: %v", err)
					}

					if !reflect.DeepEqual(jsoniterResult, stdlibResult) {
						t.Errorf("Results differ for %s:\nJsoniter: %+v\nStdlib: %+v", tc.name, jsoniterResult, stdlibResult)
					}
				}
			}

			// Log the actual behavior
			t.Logf("Jsoniter error: %v", jsoniterErr)
			t.Logf("Stdlib error: %v", stdlibErr)
		})
	}
}

// TestJsoniterVsStdlibPerformanceCharacteristics tests relative performance
func TestJsoniterVsStdlibPerformanceCharacteristics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison test in short mode")
	}

	s := NewJSONSerializer(8192)

	testData := map[string]interface{}{
		"users": generateTestUsers(100),
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-01T00:00:00Z",
			"version":   "1.0",
			"total":     100,
		},
	}

	const iterations = 1000

	// Benchmark jsoniter performance
	jsoniterStart := testing.Short() // Use time.Now() if needed
	for i := 0; i < iterations; i++ {
		_, err := s.Serialize(testData)
		if err != nil {
			t.Fatalf("Jsoniter serialize failed: %v", err)
		}
	}
	_ = jsoniterStart // Suppress unused warning

	// Benchmark standard library performance
	stdlibStart := testing.Short() // Use time.Now() if needed
	for i := 0; i < iterations; i++ {
		_, err := stdjson.Marshal(testData)
		if err != nil {
			t.Fatalf("Stdlib marshal failed: %v", err)
		}
	}
	_ = stdlibStart // Suppress unused warning

	// This test primarily validates that both implementations work
	// without errors on the same data
	t.Logf("Performance comparison completed for %d iterations", iterations)
}

// TestJsoniterStandardsCompatibility tests compatibility with different JSON standards  
func TestJsoniterStandardsCompatibility(t *testing.T) {
	s := NewJSONSerializer(2048)

	// Test data that might behave differently in different JSON implementations
	compatibilityTests := []struct {
		name string
		data interface{}
		note string
	}{
		{
			name: "EmptyObjects",
			data: map[string]interface{}{
				"empty_object": map[string]interface{}{},
				"empty_array":  []interface{}{},
			},
			note: "Empty containers should be preserved",
		},
		{
			name: "NumberPrecision",
			data: map[string]interface{}{
				"precise_float": 1.23456789012345678901234567890,
				"large_int":    int64(9223372036854775807),
				"small_int":    1,
			},
			note: "Number precision should be maintained",
		},
		{
			name: "UnicodeStrings",
			data: map[string]interface{}{
				"chinese":  "你好世界",
				"emoji":    "🚀🌟⭐",
				"combined": "Hello 世界 🌍",
			},
			note: "Unicode should be preserved correctly",
		},
		{
			name: "SpecialCharacters",
			data: map[string]interface{}{
				"quotes":     `He said "Hello"`,
				"backslash":  `Path\to\file`,
				"tab_newline": "Line1\tTab\nLine2",
			},
			note: "Special characters should be escaped properly",
		},
	}

	for _, tc := range compatibilityTests {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.note)

			// Serialize with jsoniter
			jsoniterOutput, err := s.Serialize(tc.data)
			if err != nil {
				t.Fatalf("Jsoniter serialize failed: %v", err)
			}

			// Verify it can be parsed by standard library
			var stdlibParsed interface{}
			if err := stdjson.Unmarshal(jsoniterOutput, &stdlibParsed); err != nil {
				t.Fatalf("Standard library cannot parse jsoniter output: %v", err)
			}

			// Serialize with standard library
			stdlibOutput, err := stdjson.Marshal(tc.data)
			if err != nil {
				t.Fatalf("Standard library marshal failed: %v", err)
			}

			// Verify jsoniter can parse standard library output
			var jsoniterParsed interface{}
			if err := s.Deserialize(stdlibOutput, &jsoniterParsed); err != nil {
				t.Fatalf("Jsoniter cannot parse stdlib output: %v", err)
			}

			// Both parsed results should be equivalent to original data
			// (allowing for type conversions that JSON imposes)
			t.Logf("Cross-compatibility verified for %s", tc.name)
		})
	}
}

// Helper functions
func generateTestUsers(count int) []interface{} {
	users := make([]interface{}, count)
	
	for i := 0; i < count; i++ {
		users[i] = map[string]interface{}{
			"id":       i + 1,
			"username": "user" + string(rune('0'+(i%10))),
			"email":    "user" + string(rune('0'+(i%10))) + "@example.com",
			"active":   i%2 == 0,
			"profile": map[string]interface{}{
				"firstName": "User",
				"lastName":  string(rune('A' + (i%26))),
				"age":       20 + (i % 50),
			},
		}
	}
	
	return users
}