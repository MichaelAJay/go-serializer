package serializer

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

// TestJSONStreamingWrite tests streaming serialization to io.Writer
func TestJSONStreamingWrite(t *testing.T) {
	s := NewJSONSerializer(4096)

	testCases := []struct {
		name string
		data interface{}
	}{
		{
			"Simple",
			map[string]interface{}{"key": "value", "number": 42},
		},
		{
			"Complex",
			map[string]interface{}{
				"user": map[string]interface{}{
					"id":    123,
					"name":  "Test User",
					"email": "test@example.com",
					"tags":  []string{"admin", "active"},
				},
				"timestamp": "2024-01-01T00:00:00Z",
				"metadata": map[string]interface{}{
					"version": "1.0",
					"source":  "test",
				},
			},
		},
		{
			"Array",
			[]interface{}{
				map[string]interface{}{"id": 1, "name": "Item 1"},
				map[string]interface{}{"id": 2, "name": "Item 2"},
				map[string]interface{}{"id": 3, "name": "Item 3"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			// Write to stream
			if err := s.SerializeTo(&buf, tc.data); err != nil {
				t.Fatalf("SerializeTo failed: %v", err)
			}

			// Verify data was written
			if buf.Len() == 0 {
				t.Error("No data was written to buffer")
			}

			// Compare with normal serialization
			normalSerialized, err := s.Serialize(tc.data)
			if err != nil {
				t.Fatalf("Normal serialize failed: %v", err)
			}

			// Results should be equivalent (allowing for potential formatting differences)
			streamData := buf.Bytes()
			if len(streamData) == 0 {
				t.Error("Stream serialization produced no data")
			}

			// Verify both can be deserialized to the same result
			var streamResult, normalResult interface{}
			
			if err := s.Deserialize(streamData, &streamResult); err != nil {
				t.Fatalf("Failed to deserialize stream data: %v", err)
			}

			if err := s.Deserialize(normalSerialized, &normalResult); err != nil {
				t.Fatalf("Failed to deserialize normal data: %v", err)
			}

			// Results should be equivalent
			if !reflect.DeepEqual(streamResult, normalResult) {
				t.Error("Stream and normal serialization produced different results")
			}
		})
	}
}

// TestJSONStreamingRead tests streaming deserialization from io.Reader
func TestJSONStreamingRead(t *testing.T) {
	s := NewJSONSerializer(4096)

	testData := map[string]interface{}{
		"message": "Hello, World!",
		"number":  42,
		"active":  true,
		"items":   []string{"a", "b", "c"},
		"nested": map[string]interface{}{
			"key":   "value",
			"count": 10,
		},
	}

	// First serialize the data
	serialized, err := s.Serialize(testData)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Create reader from serialized data
	reader := bytes.NewReader(serialized)

	// Deserialize from stream
	var result map[string]interface{}
	if err := s.DeserializeFrom(reader, &result); err != nil {
		t.Fatalf("DeserializeFrom failed: %v", err)
	}

	// Verify content
	if result["message"] != "Hello, World!" {
		t.Errorf("Expected message 'Hello, World!', got %v", result["message"])
	}

	if result["number"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected number 42, got %v", result["number"])
	}

	if result["active"] != true {
		t.Errorf("Expected active true, got %v", result["active"])
	}
}

// TestJSONStreamingLargeData tests streaming with large datasets
func TestJSONStreamingLargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large data streaming test in short mode")
	}

	s := NewJSONSerializer(64 * 1024)

	// Create large dataset
	const numRecords = 10000
	largeData := map[string]interface{}{
		"records": generateLargeRecordSet(numRecords),
		"metadata": map[string]interface{}{
			"total":     numRecords,
			"timestamp": "2024-01-01T00:00:00Z",
		},
	}

	// Test streaming serialization
	var buf bytes.Buffer
	if err := s.SerializeTo(&buf, largeData); err != nil {
		t.Fatalf("SerializeTo failed for large data: %v", err)
	}

	t.Logf("Large data serialized to %d bytes", buf.Len())

	// Test streaming deserialization
	reader := bytes.NewReader(buf.Bytes())
	var result map[string]interface{}
	if err := s.DeserializeFrom(reader, &result); err != nil {
		t.Fatalf("DeserializeFrom failed for large data: %v", err)
	}

	// Verify structure
	if result["metadata"].(map[string]interface{})["total"] != float64(numRecords) {
		t.Error("Large data deserialization failed - metadata incorrect")
	}

	records := result["records"].([]interface{})
	if len(records) != numRecords {
		t.Errorf("Expected %d records, got %d", numRecords, len(records))
	}
}

// TestJSONStreamingIOErrors tests error handling with faulty readers/writers
func TestJSONStreamingIOErrors(t *testing.T) {
	s := NewJSONSerializer(1024)

	testData := map[string]interface{}{
		"key": "value",
		"num": 42,
	}

	t.Run("WriterError", func(t *testing.T) {
		// Create a writer that always fails
		errorWriter := &failingWriter{failAfter: 0}

		err := s.SerializeTo(errorWriter, testData)
		if err == nil {
			t.Error("Expected error when writing to failing writer")
		}
	})

	t.Run("WriterFailsPartway", func(t *testing.T) {
		// Writer that fails after writing some data
		errorWriter := &failingWriter{failAfter: 10}

		err := s.SerializeTo(errorWriter, testData)
		if err == nil {
			t.Error("Expected error when writer fails partway through")
		}
	})

	t.Run("ReaderError", func(t *testing.T) {
		// Create a reader that always fails
		errorReader := &failingReader{failAfter: 0}

		var result map[string]interface{}
		err := s.DeserializeFrom(errorReader, &result)
		if err == nil {
			t.Error("Expected error when reading from failing reader")
		}
	})

	t.Run("ReaderFailsPartway", func(t *testing.T) {
		// Serialize some data first
		var buf bytes.Buffer
		s.SerializeTo(&buf, testData)

		// Create reader that fails after reading some data
		errorReader := &failingReader{
			data:      buf.Bytes(),
			failAfter: 5,
		}

		var result map[string]interface{}
		err := s.DeserializeFrom(errorReader, &result)
		if err == nil {
			t.Error("Expected error when reader fails partway through")
		}
	})

	t.Run("NilWriter", func(t *testing.T) {
		err := s.SerializeTo(nil, testData)
		if err == nil {
			t.Error("Expected error when writer is nil")
		}
	})

	t.Run("NilReader", func(t *testing.T) {
		var result map[string]interface{}
		err := s.DeserializeFrom(nil, &result)
		if err == nil {
			t.Error("Expected error when reader is nil")
		}
	})
}

// TestJSONStreamingPartialData tests handling of partial/incomplete data
func TestJSONStreamingPartialData(t *testing.T) {
	s := NewJSONSerializer(1024)

	testCases := []struct {
		name         string
		partialJSON  string
		expectError  bool
	}{
		{"Empty", "", true},
		{"PartialObject", `{"key": "val`, true},
		{"PartialString", `{"key": "value", "other": "unfinished`, true},
		{"PartialArray", `{"items": [1, 2`, true},
		{"ValidButIncomplete", `{"key": "value"`, true}, // Missing closing brace
		{"TruncatedNested", `{"outer": {"inner"`, true},
		{"CompleteValid", `{"key": "value"}`, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.partialJSON)

			var result map[string]interface{}
			err := s.DeserializeFrom(reader, &result)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for partial JSON %q, but got success", tc.partialJSON)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for valid JSON %q: %v", tc.partialJSON, err)
				}
			}
		})
	}
}

// TestJSONStreamingMultipleObjects tests streaming with multiple JSON objects
func TestJSONStreamingMultipleObjects(t *testing.T) {
	s := NewJSONSerializer(2048)

	objects := []interface{}{
		map[string]interface{}{"id": 1, "name": "First"},
		map[string]interface{}{"id": 2, "name": "Second"},
		map[string]interface{}{"id": 3, "name": "Third"},
	}

	// Serialize each object to the same buffer (as separate JSON documents)
	var buf bytes.Buffer
	for _, obj := range objects {
		if err := s.SerializeTo(&buf, obj); err != nil {
			t.Fatalf("SerializeTo failed: %v", err)
		}
	}

	// The buffer now contains multiple JSON documents
	// Standard JSON decoder should read them one by one
	reader := bytes.NewReader(buf.Bytes())

	for i, expectedObj := range objects {
		var result map[string]interface{}
		if err := s.DeserializeFrom(reader, &result); err != nil {
			t.Fatalf("DeserializeFrom failed for object %d: %v", i, err)
		}

		expectedMap := expectedObj.(map[string]interface{})
		if result["id"] != float64(expectedMap["id"].(int)) {
			t.Errorf("Object %d: expected id %v, got %v", i, expectedMap["id"], result["id"])
		}

		if result["name"] != expectedMap["name"] {
			t.Errorf("Object %d: expected name %v, got %v", i, expectedMap["name"], result["name"])
		}
	}
}

// TestJSONStreamingEncodingOptions tests that streaming respects encoding options
func TestJSONStreamingEncodingOptions(t *testing.T) {
	s := NewJSONSerializer(1024)

	// Test data with HTML that should not be escaped
	testData := map[string]interface{}{
		"html":    "<script>alert('test')</script>",
		"message": "Hello & welcome to the \"test\"",
	}

	var buf bytes.Buffer
	if err := s.SerializeTo(&buf, testData); err != nil {
		t.Fatalf("SerializeTo failed: %v", err)
	}

	output := buf.String()

	// HTML should not be escaped (SetEscapeHTML(false) behavior)
	if !strings.Contains(output, "<script>") {
		t.Error("Expected unescaped HTML tags in stream output")
	}

	if strings.Contains(output, "&lt;") || strings.Contains(output, "\\u003c") {
		t.Error("HTML should not be escaped in stream output")
	}
}

// Benchmark functions for streaming performance
func BenchmarkJSONStreamingWrite(b *testing.B) {
	s := NewJSONSerializer(8192)

	testData := map[string]interface{}{
		"users": generateLargeRecordSet(1000),
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-01T00:00:00Z",
			"version":   "1.0",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := s.SerializeTo(&buf, testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONStreamingRead(b *testing.B) {
	s := NewJSONSerializer(8192)

	// Pre-serialize test data
	testData := map[string]interface{}{
		"users": generateLargeRecordSet(1000),
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-01T00:00:00Z",
			"version":   "1.0",
		},
	}

	var buf bytes.Buffer
	s.SerializeTo(&buf, testData)
	serializedData := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(serializedData)
		var result map[string]interface{}
		err := s.DeserializeFrom(reader, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONStreamingVsNormal(b *testing.B) {
	s := NewJSONSerializer(4096)

	testData := map[string]interface{}{
		"message": "benchmark test",
		"items":   []string{"a", "b", "c", "d", "e"},
		"metadata": map[string]interface{}{
			"count": 5,
			"type":  "test",
		},
	}

	b.Run("Streaming", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := s.SerializeTo(&buf, testData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Normal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := s.Serialize(testData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Helper functions and types
func generateLargeRecordSet(count int) []interface{} {
	records := make([]interface{}, count)
	for i := 0; i < count; i++ {
		records[i] = map[string]interface{}{
			"id":       i,
			"name":     "Record " + string(rune('A'+(i%26))),
			"value":    float64(i) * 1.23,
			"active":   i%2 == 0,
			"created":  "2024-01-01T00:00:00Z",
			"tags":     []string{"tag" + string(rune('0'+(i%10)))},
		}
	}
	return records
}

type failingWriter struct {
	written   int
	failAfter int
}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	if w.written >= w.failAfter {
		return 0, errors.New("writer failure")
	}

	writeCount := len(p)
	if w.written+writeCount > w.failAfter {
		writeCount = w.failAfter - w.written
	}

	w.written += writeCount
	if writeCount < len(p) {
		return writeCount, errors.New("writer failure")
	}

	return writeCount, nil
}

type failingReader struct {
	data      []byte
	pos       int
	failAfter int
}

func (r *failingReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.failAfter {
		return 0, errors.New("reader failure")
	}

	if r.data == nil || r.pos >= len(r.data) {
		return 0, io.EOF
	}

	readCount := len(p)
	remaining := len(r.data) - r.pos
	if readCount > remaining {
		readCount = remaining
	}

	if r.pos+readCount > r.failAfter {
		readCount = r.failAfter - r.pos
	}

	copy(p, r.data[r.pos:r.pos+readCount])
	r.pos += readCount

	if r.pos >= r.failAfter && readCount > 0 {
		return readCount, errors.New("reader failure")
	}

	return readCount, nil
}

