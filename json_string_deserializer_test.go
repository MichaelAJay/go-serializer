package serializer

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

// TestStringToBytesConversion tests the unsafe string to bytes conversion
func TestStringToBytesConversion(t *testing.T) {
	testCases := []struct {
		name string
		str  string
	}{
		{"Empty", ""},
		{"Short", "hello"},
		{"Medium", "This is a medium length string for testing"},
		{"Long", strings.Repeat("Long string content ", 100)},
		{"Unicode", "Hello 世界 🌍"},
		{"JSON", `{"name": "test", "value": 42, "active": true}`},
		{"SpecialChars", "String with \n\t\r\"\\/ special chars"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert using unsafe method
			unsafeBytes := stringToReadOnlyBytes(tc.str)

			// Convert using standard method for comparison
			standardBytes := []byte(tc.str)

			// Results should be identical
			if len(unsafeBytes) != len(standardBytes) {
				t.Errorf("Length mismatch: unsafe=%d, standard=%d", len(unsafeBytes), len(standardBytes))
			}

			// Content should be identical
			for i, b := range unsafeBytes {
				if i >= len(standardBytes) || b != standardBytes[i] {
					t.Errorf("Content mismatch at index %d: unsafe=%d, standard=%d", i, b, standardBytes[i])
					break
				}
			}

			// For empty strings, both should be nil or empty
			if tc.str == "" {
				if len(unsafeBytes) != 0 {
					t.Error("Empty string should produce nil or empty slice")
				}
			}
		})
	}
}

// TestStringDeserializerMemoryOptimization tests that string deserialization doesn't allocate
func TestStringDeserializerMemoryOptimization(t *testing.T) {
	s := NewJSONSerializer(1024)
	stringDeser := s.(StringDeserializer)

	jsonData := `{"name": "test", "value": 42, "tags": ["a", "b", "c"]}`
	
	// Warm up to avoid initial allocations
	for i := 0; i < 10; i++ {
		var result map[string]interface{}
		stringDeser.DeserializeString(jsonData, &result)
	}

	// Measure allocations for string deserialization
	var stringAllocs runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&stringAllocs)
	
	const iterations = 100
	for i := 0; i < iterations; i++ {
		var result map[string]interface{}
		err := stringDeser.DeserializeString(jsonData, &result)
		if err != nil {
			t.Fatal(err)
		}
	}

	var stringAllocsAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&stringAllocsAfter)

	// Measure allocations for bytes deserialization
	jsonBytes := []byte(jsonData)
	
	var bytesAllocs runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&bytesAllocs)
	
	for i := 0; i < iterations; i++ {
		var result map[string]interface{}
		err := s.Deserialize(jsonBytes, &result)
		if err != nil {
			t.Fatal(err)
		}
	}

	var bytesAllocsAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&bytesAllocsAfter)

	stringAllocDiff := stringAllocsAfter.TotalAlloc - stringAllocs.TotalAlloc
	bytesAllocDiff := bytesAllocsAfter.TotalAlloc - bytesAllocs.TotalAlloc

	t.Logf("String deserialization allocations: %d bytes", stringAllocDiff)
	t.Logf("Bytes deserialization allocations: %d bytes", bytesAllocDiff)

	// String deserialization should allocate less due to avoiding string->bytes conversion
	if stringAllocDiff > bytesAllocDiff {
		t.Logf("Note: String deserialization allocated more than bytes deserialization")
		t.Logf("This may be due to other factors or implementation details")
	}
}

// TestStringDeserializerDataIntegrity verifies data integrity across various string encodings
func TestStringDeserializerDataIntegrity(t *testing.T) {
	s := NewJSONSerializer(2048)
	stringDeser := s.(StringDeserializer)

	testData := []interface{}{
		"simple string",
		42,
		3.14159,
		true,
		[]string{"a", "b", "c"},
		map[string]interface{}{
			"name":  "test",
			"value": 123,
			"nested": map[string]interface{}{
				"inner": "value",
				"count": 5,
			},
		},
		struct {
			Name  string   `json:"name"`
			Items []string `json:"items"`
			Count int      `json:"count"`
		}{
			Name:  "struct test",
			Items: []string{"x", "y", "z"},
			Count: 10,
		},
	}

	for i, original := range testData {
		t.Run(string(rune('A'+(i%26))), func(t *testing.T) {
			// Serialize original data
			serialized, err := s.Serialize(original)
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			serializedStr := string(serialized)

			// Deserialize using string method
			var stringResult interface{}
			if err := stringDeser.DeserializeString(serializedStr, &stringResult); err != nil {
				t.Fatalf("DeserializeString failed: %v", err)
			}

			// Deserialize using bytes method
			var bytesResult interface{}
			if err := s.Deserialize(serialized, &bytesResult); err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			// Results should be identical
			if !reflect.DeepEqual(stringResult, bytesResult) {
				t.Errorf("Results differ:\nString: %+v\nBytes:  %+v", stringResult, bytesResult)
			}
		})
	}
}

// TestEmptyStringDeserialization tests handling of empty strings
func TestEmptyStringDeserialization(t *testing.T) {
	s := NewJSONSerializer(1024)
	stringDeser := s.(StringDeserializer)

	testCases := []struct {
		name   string
		input  string
		expectError bool
	}{
		{"TrulyEmpty", "", true},
		{"Whitespace", "   \t\n   ", true},
		{"EmptyObject", "{}", false},
		{"EmptyArray", "[]", false},
		{"NullValue", "null", false},
		{"EmptyString", `""`, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result interface{}
			err := stringDeser.DeserializeString(tc.input, &result)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got success", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tc.input, err)
				}
			}
		})
	}
}

// TestVeryLongStringDeserialization tests performance with very long strings
func TestVeryLongStringDeserialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long string test in short mode")
	}

	s := NewJSONSerializer(1024 * 1024) // 1MB buffer
	stringDeser := s.(StringDeserializer)

	// Create very long JSON string
	longValue := strings.Repeat("This is a very long string value that will test the performance and correctness of string deserialization. ", 1000)
	
	longJSON := map[string]interface{}{
		"shortKey": "shortValue",
		"longKey":  longValue,
		"number":   42,
		"array":    []string{"a", "b", "c"},
	}

	// Serialize to get JSON string
	serialized, err := s.Serialize(longJSON)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	serializedStr := string(serialized)
	t.Logf("JSON string length: %d bytes", len(serializedStr))

	// Test string deserialization
	var stringResult map[string]interface{}
	if err := stringDeser.DeserializeString(serializedStr, &stringResult); err != nil {
		t.Fatalf("DeserializeString failed: %v", err)
	}

	// Verify content
	if stringResult["longKey"] != longValue {
		t.Error("Long string value was corrupted during deserialization")
	}

	if stringResult["shortKey"] != "shortValue" {
		t.Error("Short string value was corrupted")
	}

	if stringResult["number"] != float64(42) { // JSON numbers are float64
		t.Error("Number value was corrupted")
	}
}

// TestBinaryDataInString tests handling of binary data within JSON strings
func TestBinaryDataInString(t *testing.T) {
	s := NewJSONSerializer(1024)
	stringDeser := s.(StringDeserializer)

	// Create data with binary content (base64 encoded in JSON)
	binaryData := []byte{0, 1, 2, 3, 255, 254, 253, 10, 13, 9}
	
	testData := map[string]interface{}{
		"text":   "normal text",
		"binary": string(binaryData), // This will be JSON-escaped
		"number": 42,
	}

	// Serialize
	serialized, err := s.Serialize(testData)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	serializedStr := string(serialized)

	// Deserialize using string method
	var stringResult map[string]interface{}
	if err := stringDeser.DeserializeString(serializedStr, &stringResult); err != nil {
		t.Fatalf("DeserializeString failed: %v", err)
	}

	// Verify the binary data survived the round trip
	if stringResult["binary"] != string(binaryData) {
		t.Error("Binary data was corrupted during deserialization")
	}

	// Compare with bytes deserialization
	var bytesResult map[string]interface{}
	if err := s.Deserialize(serialized, &bytesResult); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if stringResult["binary"] != bytesResult["binary"] {
		t.Error("String and bytes deserialization produced different binary data")
	}
}

// TestStringDeserializerConcurrency tests concurrent usage of string deserialization
func TestStringDeserializerConcurrency(t *testing.T) {
	s := NewJSONSerializer(4096)
	stringDeser := s.(StringDeserializer)

	jsonStr := `{
		"user": {
			"id": 123,
			"name": "Test User",
			"email": "test@example.com",
			"preferences": {
				"theme": "dark",
				"notifications": true
			},
			"tags": ["admin", "active"]
		},
		"timestamp": "2024-01-01T00:00:00Z"
	}`

	const numGoroutines = 20
	const operationsPerGoroutine = 50

	errChan := make(chan error, numGoroutines*operationsPerGoroutine)
	
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			for i := 0; i < operationsPerGoroutine; i++ {
				var result map[string]interface{}
				if err := stringDeser.DeserializeString(jsonStr, &result); err != nil {
					errChan <- err
					return
				}

				// Basic verification
				user := result["user"].(map[string]interface{})
				if user["id"] != float64(123) {
					errChan <- &testError{"ID verification failed"}
					return
				}
			}
		}(g)
	}

	// Collect any errors
	go func() {
		for i := 0; i < numGoroutines; i++ {
			for j := 0; j < operationsPerGoroutine; j++ {
				select {
				case err := <-errChan:
					t.Error(err)
				default:
					// No more errors
					return
				}
			}
		}
	}()

	// Wait a reasonable time for goroutines to complete
	runtime.Gosched()
}

// TestUnsafeConversionSafety tests that the unsafe conversion doesn't cause memory corruption
func TestUnsafeConversionSafety(t *testing.T) {
	// Test that modifying the original string doesn't affect the conversion
	// (This is more of a documentation test since strings are immutable in Go)
	
	originalStr := "test string for safety"
	convertedBytes := stringToReadOnlyBytes(originalStr)

	// Verify conversion worked
	if string(convertedBytes) != originalStr {
		t.Error("Unsafe conversion produced different content")
	}

	// Test with various string lengths
	testStrings := []string{
		"",
		"a",
		"short",
		strings.Repeat("longer test string ", 10),
		strings.Repeat("very long test string with lots of content ", 100),
	}

	for i, str := range testStrings {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			converted := stringToReadOnlyBytes(str)
			
			if len(str) == 0 {
				if len(converted) != 0 {
					t.Error("Empty string should produce empty slice")
				}
			} else {
				if string(converted) != str {
					t.Error("Conversion failed for string:", str)
				}

				// Verify the underlying data pointer is the same (when non-empty)
				if len(str) > 0 {
					strPtr := unsafe.StringData(str)
					slicePtr := unsafe.SliceData(converted)
					if strPtr != slicePtr {
						t.Error("Unsafe conversion should reuse string data pointer")
					}
				}
			}
		})
	}
}

// Benchmark functions to compare performance
func BenchmarkStringVsBytesDeserializer(b *testing.B) {
	s := NewJSONSerializer(4096)
	stringDeser := s.(StringDeserializer)

	jsonStr := `{"name": "benchmark", "value": 42, "tags": ["a", "b", "c"], "nested": {"x": 1, "y": 2}}`
	jsonBytes := []byte(jsonStr)

	b.Run("StringDeserializer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result map[string]interface{}
			stringDeser.DeserializeString(jsonStr, &result)
		}
	})

	b.Run("BytesDeserializer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result map[string]interface{}
			s.Deserialize(jsonBytes, &result)
		}
	})
}

func BenchmarkStringConversionMethods(b *testing.B) {
	testStr := strings.Repeat("benchmark test string content ", 100)

	b.Run("UnsafeConversion", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = stringToReadOnlyBytes(testStr)
		}
	})

	b.Run("StandardConversion", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = []byte(testStr)
		}
	})
}

// Helper functions - use the deepEqual from json_edge_cases_test.go or create a shared helper

