package serializer

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestJSONLargeNumbers tests serialization of very large numbers
func TestJSONLargeNumbers(t *testing.T) {
	s := NewJSONSerializer(1024)

	testCases := []struct {
		name   string
		value  interface{}
		expectError bool
	}{
		{"MaxInt64", int64(math.MaxInt64), false},
		{"MinInt64", int64(math.MinInt64), false},
		{"MaxFloat64", math.MaxFloat64, false},
		{"MinFloat64", -math.MaxFloat64, false},
		{"SmallestFloat64", math.SmallestNonzeroFloat64, false},
		{"PosInf", math.Inf(1), true},
		{"NegInf", math.Inf(-1), true},
		{"NaN", math.NaN(), true},
		{"LargeInt", "123456789012345678901234567890", false}, // Large number as string
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]interface{}{"number": tc.value}

			serialized, err := s.Serialize(data)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got success", tc.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("Serialize failed for %s: %v", tc.name, err)
			}

			// Test deserialization
			var result map[string]interface{}
			if err := s.Deserialize(serialized, &result); err != nil {
				t.Fatalf("Deserialize failed for %s: %v", tc.name, err)
			}

			// Verify the number exists
			if result["number"] == nil {
				t.Errorf("Number field is nil after round-trip for %s", tc.name)
			}
		})
	}
}

// TestJSONUnicodeStrings tests Unicode and special character handling
func TestJSONUnicodeStrings(t *testing.T) {
	s := NewJSONSerializer(2048)

	testCases := []struct {
		name string
		text string
	}{
		{"ASCII", "Hello World"},
		{"BasicUnicode", "Héllo Wørld"},
		{"Emoji", "Hello 🌍 World 🎉"},
		{"Chinese", "你好世界"},
		{"Japanese", "こんにちは世界"},
		{"Arabic", "مرحبا بالعالم"},
		{"Russian", "Привет мир"},
		{"Mixed", "Hello 世界 🌍 Привет مرحبا"},
		{"ControlChars", "Text with \t tab and \n newline"},
		{"Quotes", `Text with "quotes" and 'apostrophes'`},
		{"Backslashes", `Path\to\file and C:\Windows\System32`},
		{"ZeroWidth", "Text\u200Bwith\u200Czero\u200Dwidth\uFEFFchars"},
		{"Surrogate", "High surrogate: 😀"}, // Emoji (was surrogate pair)
		{"LongUnicode", strings.Repeat("🚀", 100)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]interface{}{"text": tc.text}

			// Serialize
			serialized, err := s.Serialize(data)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Deserialize
			var result map[string]interface{}
			if err := s.Deserialize(serialized, &result); err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			// Verify content
			if result["text"] != tc.text {
				t.Errorf("Text mismatch: expected %q, got %q", tc.text, result["text"])
			}
		})
	}
}

// TestJSONSpecialCharacters tests handling of special characters that might cause issues
func TestJSONSpecialCharacters(t *testing.T) {
	s := NewJSONSerializer(1024)

	testCases := []struct {
		name string
		data interface{}
	}{
		{"NullBytes", map[string]string{"data": "before\x00after"}},
		{"BinaryData", map[string]string{"data": string([]byte{0, 1, 2, 3, 255, 254, 253})}},
		{"HighBitSet", map[string]string{"data": string([]byte{128, 129, 255})}},
		{"JSONEscapes", map[string]string{"data": "\b\f\n\r\t\"\\"}},
		{"HTMLTags", map[string]string{"html": "<script>alert('test')</script>"}},
		{"SQLInjection", map[string]string{"sql": "'; DROP TABLE users; --"}},
		{"PathTraversal", map[string]string{"path": "../../../etc/passwd"}},
		{"LongString", map[string]string{"long": strings.Repeat("A", 10000)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize
			serialized, err := s.Serialize(tc.data)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Deserialize
			var result interface{}
			if err := s.Deserialize(serialized, &result); err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			// Basic verification that structure is maintained
			if result == nil {
				t.Error("Result should not be nil")
			}
		})
	}
}

// TestJSONDeepNesting tests very deeply nested objects
func TestJSONDeepNesting(t *testing.T) {
	s := NewJSONSerializer(64 * 1024)

	// Test various nesting depths
	testCases := []struct {
		name  string
		depth int
		expectError bool
	}{
		{"Shallow", 10, false},
		{"Medium", 50, false},
		{"Deep", 100, false},
		{"VeryDeep", 500, false},
		{"ExtremelyDeep", 1000, false}, // May hit stack limits
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create deeply nested object
			data := createNestedObject(tc.depth)

			// Serialize
			serialized, err := s.Serialize(data)
			
			if tc.expectError {
				if err == nil {
					t.Error("Expected error for extremely deep nesting")
				}
				return
			}

			if err != nil {
				t.Fatalf("Serialize failed at depth %d: %v", tc.depth, err)
			}

			// Deserialize
			var result interface{}
			if err := s.Deserialize(serialized, &result); err != nil {
				t.Fatalf("Deserialize failed at depth %d: %v", tc.depth, err)
			}

			// Verify structure is intact
			depth := measureNestingDepth(result)
			if depth < tc.depth-1 { // Allow for some variance
				t.Errorf("Expected depth around %d, got %d", tc.depth, depth)
			}
		})
	}
}

// TestJSONCircularReferences tests detection and handling of circular references
func TestJSONCircularReferences(t *testing.T) {
	s := NewJSONSerializer(1024)

	// Create circular reference structures
	testCases := []struct {
		name string
		createData func() interface{}
	}{
		{
			name: "MapCircular",
			createData: func() interface{} {
				m := make(map[string]interface{})
				m["self"] = m
				return m
			},
		},
		{
			name: "SliceCircular",
			createData: func() interface{} {
				s := make([]interface{}, 1)
				s[0] = s
				return s
			},
		},
		{
			name: "IndirectCircular",
			createData: func() interface{} {
				a := make(map[string]interface{})
				b := make(map[string]interface{})
				a["b"] = b
				b["a"] = a
				return a
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.createData()

			// This should fail - JSON cannot represent circular references
			_, err := s.Serialize(data)
			if err == nil {
				t.Error("Expected error for circular reference, got success")
			}

			// Error should mention the circular reference or stack overflow
			errStr := err.Error()
			if !strings.Contains(strings.ToLower(errStr), "circular") && 
			   !strings.Contains(strings.ToLower(errStr), "cycle") &&
			   !strings.Contains(strings.ToLower(errStr), "stack") {
				t.Logf("Error message: %s", errStr)
				// Don't fail - different JSON libraries may have different error messages
			}
		})
	}
}

// TestJSONMemoryUsage tests behavior with various memory usage patterns
func TestJSONMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	s := NewJSONSerializer(32 * 1024)

	testCases := []struct {
		name string
		size int
	}{
		{"Small", 100},
		{"Medium", 10000},
		{"Large", 100000},
		{"Huge", 1000000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create large data structure
			data := make(map[string]interface{})
			for i := 0; i < tc.size; i++ {
				key := "key_" + strconv.Itoa(i)
				data[key] = map[string]interface{}{
					"id":    i,
					"name":  "Item " + strconv.Itoa(i),
					"value": float64(i) * 1.23,
					"tags":  []string{"tag1", "tag2"},
				}
			}

			// Serialize
			serialized, err := s.Serialize(data)
			if err != nil {
				t.Fatalf("Serialize failed for size %d: %v", tc.size, err)
			}

			if len(serialized) == 0 {
				t.Error("Serialized data is empty")
			}

			// Deserialize
			var result map[string]interface{}
			if err := s.Deserialize(serialized, &result); err != nil {
				t.Fatalf("Deserialize failed for size %d: %v", tc.size, err)
			}

			// Basic verification
			if len(result) != tc.size {
				t.Errorf("Expected %d items, got %d", tc.size, len(result))
			}
		})
	}
}

// TestJSONLargeDatasets tests handling of large datasets
func TestJSONLargeDatasets(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	s := NewJSONSerializer(1024 * 1024) // 1MB buffer

	// Create large dataset
	const numRecords = 50000
	records := make([]map[string]interface{}, numRecords)
	
	for i := 0; i < numRecords; i++ {
		records[i] = map[string]interface{}{
			"id":          i,
			"uuid":        "uuid-" + strconv.Itoa(i),
			"name":        "Record " + strconv.Itoa(i),
			"description": "This is a description for record number " + strconv.Itoa(i),
			"value":       float64(i) * 3.14159,
			"active":      i%2 == 0,
			"tags":        []string{"tag" + strconv.Itoa(i%10), "category" + strconv.Itoa(i%5)},
			"metadata": map[string]interface{}{
				"created": "2024-01-01T00:00:00Z",
				"updated": "2024-01-02T00:00:00Z",
			},
		}
	}

	dataset := map[string]interface{}{
		"records": records,
		"total":   numRecords,
		"version": "1.0",
	}

	// Serialize
	serialized, err := s.Serialize(dataset)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	t.Logf("Serialized size: %d bytes", len(serialized))

	// Deserialize
	var result map[string]interface{}
	if err := s.Deserialize(serialized, &result); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	// Verify
	if result["total"] != float64(numRecords) { // JSON numbers are float64
		t.Errorf("Expected total %d, got %v", numRecords, result["total"])
	}

	resultRecords := result["records"].([]interface{})
	if len(resultRecords) != numRecords {
		t.Errorf("Expected %d records, got %d", numRecords, len(resultRecords))
	}
}

// TestJSONGoroutineSafety tests concurrent access safety
func TestJSONGoroutineSafety(t *testing.T) {
	s := NewJSONSerializer(4096)

	const numGoroutines = 20
	const operationsPerGoroutine = 100

	testData := map[string]interface{}{
		"message": "Hello, World!",
		"number":  42,
		"array":   []interface{}{1, 2, 3, "test"},
		"object": map[string]interface{}{
			"nested": "value",
			"count":  10,
		},
	}

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*operationsPerGoroutine)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < operationsPerGoroutine; i++ {
				// Add goroutine-specific data
				data := make(map[string]interface{})
				for k, v := range testData {
					data[k] = v
				}
				data["goroutine"] = goroutineID
				data["operation"] = i

				// Serialize
				serialized, err := s.Serialize(data)
				if err != nil {
					errChan <- err
					return
				}

				// Deserialize
				var result map[string]interface{}
				if err := s.Deserialize(serialized, &result); err != nil {
					errChan <- err
					return
				}

				// Verify key fields
				if result["goroutine"] != float64(goroutineID) {
					errChan <- &testError{"goroutine ID mismatch"}
					return
				}
				if result["operation"] != float64(i) {
					errChan <- &testError{"operation ID mismatch"}
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Error(err)
	}
}

// TestJSONMalformedInputHandling tests various types of malformed input
func TestJSONMalformedInputHandling(t *testing.T) {
	s := NewJSONSerializer(1024)

	malformedInputs := []struct {
		name  string
		input string
	}{
		{"Empty", ""},
		{"JustSpaces", "   \t\n   "},
		{"UnterminatedString", `{"key": "value`},
		{"UnterminatedObject", `{"key": "value", "other"`},
		{"UnterminatedArray", `[1, 2, 3`},
		{"TrailingComma", `{"a": 1, "b": 2,}`},
		{"ExtraComma", `{"a": 1,, "b": 2}`},
		{"SingleQuotes", `{'key': 'value'}`},
		{"UnquotedKeys", `{key: "value"}`},
		{"InvalidEscape", `{"key": "\\x"}`},
		{"InvalidUnicode", `{"key": "\\uXXXX"}`},
		{"MixedTypes", `{"key": undefined}`},
		{"Comments", `{"key": "value", // comment}`},
		{"TrailingText", `{"key": "value"} extra`},
		{"InvalidNumber", `{"key": 01.23}`},
		{"HexNumber", `{"key": 0xFF}`},
		{"BinaryData", string([]byte{0xFF, 0xFE, 0xFD, 0x01, 0x02})},
	}

	for _, tc := range malformedInputs {
		t.Run(tc.name, func(t *testing.T) {
			var result interface{}
			err := s.Deserialize([]byte(tc.input), &result)

			// All these should fail
			if err == nil {
				t.Errorf("Expected error for malformed input %q, but got success with result: %+v", tc.input, result)
			}
		})
	}
}

// Helper functions
func createNestedObject(depth int) interface{} {
	if depth <= 0 {
		return map[string]interface{}{
			"leaf": true,
			"value": "bottom",
		}
	}

	return map[string]interface{}{
		"level": depth,
		"child": createNestedObject(depth - 1),
		"data":  "level_" + strconv.Itoa(depth),
	}
}

func measureNestingDepth(obj interface{}) int {
	switch v := obj.(type) {
	case map[string]interface{}:
		maxDepth := 0
		for _, val := range v {
			depth := 1 + measureNestingDepth(val)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
		return maxDepth
	case []interface{}:
		maxDepth := 0
		for _, val := range v {
			depth := 1 + measureNestingDepth(val)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
		return maxDepth
	default:
		return 0
	}
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}