package serializer

import (
	stdjson "encoding/json"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// Test data generators for benchmarks
func generateSmallObject() interface{} {
	return map[string]interface{}{
		"id":   42,
		"name": "test",
		"active": true,
	}
}

func generateMediumObject() interface{} {
	return map[string]interface{}{
		"user": map[string]interface{}{
			"id":       12345,
			"username": "testuser",
			"email":    "test@example.com",
			"profile": map[string]interface{}{
				"firstName": "Test",
				"lastName":  "User",
				"age":       30,
				"settings": map[string]interface{}{
					"theme":         "dark",
					"notifications": true,
					"language":      "en",
				},
			},
		},
		"metadata": map[string]interface{}{
			"created":    "2024-01-01T00:00:00Z",
			"updated":    "2024-01-01T12:00:00Z",
			"version":    "1.0.0",
			"tags":       []string{"user", "active", "premium"},
		},
	}
}

func generateLargeObject() interface{} {
	users := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		users[i] = map[string]interface{}{
			"id":       i,
			"username": "user" + string(rune('0'+(i%10))),
			"email":    "user" + string(rune('0'+(i%10))) + "@example.com",
			"profile": map[string]interface{}{
				"firstName": "User",
				"lastName":  string(rune('A'+(i%26))),
				"age":       20 + (i % 50),
				"address": map[string]interface{}{
					"street":  "123 Test St",
					"city":    "Test City",
					"state":   "TS",
					"zip":     "12345",
					"country": "TestLand",
				},
			},
			"preferences": map[string]interface{}{
				"theme":         []string{"dark", "light"}[i%2],
				"notifications": i%3 == 0,
				"language":      "en",
				"timezone":      "UTC",
			},
			"tags": []string{"user", "active"},
		}
	}

	return map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"total":   100,
			"page":    1,
			"perPage": 100,
			"pages":   1,
		},
		"metadata": map[string]interface{}{
			"generated":  "2024-01-01T00:00:00Z",
			"version":    "2.0.0",
			"apiVersion": "v1",
		},
	}
}

func generateNestedStructure(depth int) interface{} {
	if depth <= 0 {
		return map[string]interface{}{
			"value": "leaf",
			"id":    depth,
		}
	}

	return map[string]interface{}{
		"level":    depth,
		"data":     "level_" + string(rune('0'+depth%10)),
		"children": []interface{}{
			generateNestedStructure(depth - 1),
			generateNestedStructure(depth - 1),
		},
		"metadata": map[string]interface{}{
			"depth": depth,
			"type":  "branch",
		},
	}
}

// Core serialization benchmarks
func BenchmarkJSONSerialize(b *testing.B) {
	s := NewJSONSerializer(32 * 1024)

	testCases := []struct {
		name string
		data interface{}
	}{
		{"Small", generateSmallObject()},
		{"Medium", generateMediumObject()},
		{"Large", generateLargeObject()},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := s.Serialize(tc.data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkJSONDeserialize(b *testing.B) {
	s := NewJSONSerializer(32 * 1024)

	testCases := []struct {
		name string
		data interface{}
	}{
		{"Small", generateSmallObject()},
		{"Medium", generateMediumObject()},
		{"Large", generateLargeObject()},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Pre-serialize the data
			serialized, err := s.Serialize(tc.data)
			if err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var result interface{}
				err := s.Deserialize(serialized, &result)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkJSONStringDeserialize(b *testing.B) {
	s := NewJSONSerializer(32 * 1024)
	stringDeser := s.(StringDeserializer)

	testCases := []struct {
		name string
		data interface{}
	}{
		{"Small", generateSmallObject()},
		{"Medium", generateMediumObject()},
		{"Large", generateLargeObject()},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Pre-serialize the data
			serialized, err := s.Serialize(tc.data)
			if err != nil {
				b.Fatal(err)
			}
			serializedStr := string(serialized)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var result interface{}
				err := stringDeser.DeserializeString(serializedStr, &result)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Size-based benchmarks
func BenchmarkJSONSmallObjects(b *testing.B) {
	s := NewJSONSerializer(1024)

	data := generateSmallObject()

	b.Run("SerializeSmall", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := s.Serialize(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	serialized, _ := s.Serialize(data)
	b.Run("DeserializeSmall", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result interface{}
			err := s.Deserialize(serialized, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkJSONMediumObjects(b *testing.B) {
	s := NewJSONSerializer(8192)

	data := generateMediumObject()

	b.Run("SerializeMedium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := s.Serialize(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	serialized, _ := s.Serialize(data)
	b.Run("DeserializeMedium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result interface{}
			err := s.Deserialize(serialized, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkJSONLargeObjects(b *testing.B) {
	s := NewJSONSerializer(64 * 1024)

	data := generateLargeObject()

	b.Run("SerializeLarge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := s.Serialize(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	serialized, _ := s.Serialize(data)
	b.Run("DeserializeLarge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result interface{}
			err := s.Deserialize(serialized, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Data type specific benchmarks
func BenchmarkJSONNumbers(b *testing.B) {
	s := NewJSONSerializer(1024)

	numbers := map[string]interface{}{
		"int":        42,
		"int64":      int64(1234567890123456789),
		"float32":    float32(3.14159),
		"float64":    3.141592653589793,
		"scientific": 1.23e-10,
		"negative":   -987.654,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Serialize(numbers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONStrings(b *testing.B) {
	s := NewJSONSerializer(4096)

	strings := map[string]interface{}{
		"short":   "Hello",
		"medium":  "This is a medium length string for testing",
		"long":    strings.Repeat("This is a longer string for testing JSON serialization performance. ", 10),
		"unicode": "Hello 世界 🌍 Testing unicode characters",
		"escape":  "String with \"quotes\" and \\ backslashes and \n newlines",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Serialize(strings)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONArrays(b *testing.B) {
	s := NewJSONSerializer(8192)

	arrays := map[string]interface{}{
		"strings":  []string{"a", "b", "c", "d", "e"},
		"numbers":  []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		"mixed":    []interface{}{"hello", 42, true, 3.14, nil},
		"nested":   [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}},
		"large":    make([]int, 1000),
	}

	// Populate large array
	largeArray := arrays["large"].([]int)
	for i := range largeArray {
		largeArray[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Serialize(arrays)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONMaps(b *testing.B) {
	s := NewJSONSerializer(8192)

	// Generate large map
	largeMap := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		key := "key_" + string(rune('A'+(i%26))) + string(rune('0'+(i%10)))
		largeMap[key] = map[string]interface{}{
			"id":    i,
			"value": "value_" + string(rune('0'+(i%10))),
			"score": float64(i) * 1.23,
		}
	}

	maps := map[string]interface{}{
		"simple": map[string]string{"a": "1", "b": "2", "c": "3"},
		"mixed":  map[string]interface{}{"str": "hello", "num": 42, "bool": true},
		"nested": map[string]map[string]string{
			"user1": {"name": "Alice", "role": "admin"},
			"user2": {"name": "Bob", "role": "user"},
		},
		"large": largeMap,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.Serialize(maps)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONNestedStructs(b *testing.B) {
	s := NewJSONSerializer(16 * 1024)

	testCases := []struct {
		name  string
		depth int
	}{
		{"Depth5", 5},
		{"Depth10", 10},
		{"Depth15", 15},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			data := generateNestedStructure(tc.depth)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := s.Serialize(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Memory allocation benchmarks
func BenchmarkJSONAllocations(b *testing.B) {
	s := NewJSONSerializer(4096)
	data := generateMediumObject()

	b.Run("Serialize", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := s.Serialize(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	serialized, _ := s.Serialize(data)
	b.Run("Deserialize", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var result interface{}
			err := s.Deserialize(serialized, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkBufferPoolEfficiency(b *testing.B) {
	testCases := []struct {
		name          string
		maxBufferSize int
	}{
		{"NoLimit", -1},
		{"Small", 1024},
		{"Medium", 8192},
		{"Large", 32 * 1024},
	}

	data := generateMediumObject()

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			s := NewJSONSerializer(tc.maxBufferSize)
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := s.Serialize(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkStringVsBytesAllocation(b *testing.B) {
	s := NewJSONSerializer(4096)
	stringDeser := s.(StringDeserializer)

	data := generateMediumObject()
	serialized, _ := s.Serialize(data)
	serializedStr := string(serialized)

	b.Run("BytesDeserialize", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var result interface{}
			err := s.Deserialize(serialized, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("StringDeserialize", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var result interface{}
			err := stringDeser.DeserializeString(serializedStr, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Comparative benchmarks
func BenchmarkJSONvsStdlib(b *testing.B) {
	jsoniterSerializer := NewJSONSerializer(4096)
	data := generateMediumObject()

	b.Run("JsoniterSerialize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := jsoniterSerializer.Serialize(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("StdlibMarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := stdjson.Marshal(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// Deserialize comparison
	jsoniterData, _ := jsoniterSerializer.Serialize(data)
	stdlibData, _ := stdjson.Marshal(data)

	b.Run("JsoniterDeserialize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result interface{}
			err := jsoniterSerializer.Deserialize(jsoniterData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("StdlibUnmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result interface{}
			err := stdjson.Unmarshal(stdlibData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkBufferPoolvsNoPool(b *testing.B) {
	data := generateMediumObject()

	b.Run("WithPool", func(b *testing.B) {
		s := NewJSONSerializer(4096)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := s.Serialize(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		// Use standard library without pooling
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := stdjson.Marshal(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Realistic workload benchmarks
func BenchmarkJSONRealisticWorkload(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping realistic workload benchmark in short mode")
	}

	s := NewJSONSerializer(8192)

	// Generate mixed workload data
	workloadData := make([]interface{}, 100)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	for i := 0; i < 100; i++ {
		switch i % 4 {
		case 0:
			workloadData[i] = generateSmallObject()
		case 1:
			workloadData[i] = generateMediumObject()
		case 2:
			workloadData[i] = generateLargeObject()
		case 3:
			workloadData[i] = generateNestedStructure(5)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Pick random data
		data := workloadData[rng.Intn(len(workloadData))]

		// Serialize
		serialized, err := s.Serialize(data)
		if err != nil {
			b.Fatal(err)
		}

		// Deserialize (simulating complete round-trip)
		var result interface{}
		err = s.Deserialize(serialized, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}