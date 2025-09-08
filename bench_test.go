package serializer_test

import (
	"testing"
	"time"

	"github.com/MichaelAJay/go-serializer"
)

// benchmarkData contains test data of varying sizes for benchmarks
var benchmarkData = []struct {
	name string
	data any
}{
	{
		name: "SmallString",
		data: "hello world",
	},
	{
		name: "LargeString",
		data: func() string {
			large := ""
			for i := 0; i < 10000; i++ {
				large += "This is a test string for performance benchmarking. "
			}
			return large
		}(),
	},
	{
		name: "SmallStruct",
		data: struct {
			Name  string `json:"name" msgpack:"name"`
			Value int    `json:"value" msgpack:"value"`
		}{
			Name:  "test",
			Value: 42,
		},
	},
	{
		name: "LargeStruct",
		data: testStruct{
			String:    "benchmark test data",
			Int:       12345,
			Float:     3.14159,
			Bool:      true,
			Time:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Slice:     func() []string { 
				slice := make([]string, 1000)
				for i := range slice {
					slice[i] = "item"
				}
				return slice
			}(),
			Map:       func() map[string]int {
				m := make(map[string]int)
				for i := 0; i < 100; i++ {
					m[string(rune('a'+i))] = i
				}
				return m
			}(),
			Ptr:       func() *string { s := "pointer value"; return &s }(),
			Interface: "interface value",
		},
	},
}

// BenchmarkJSONDeserializeString benchmarks JSON StringDeserializer performance
func BenchmarkJSONDeserializeString(b *testing.B) {
	jsonSerializer := serializer.NewJSONSerializer()
	stringDeser := jsonSerializer.(serializer.StringDeserializer)

	for _, bd := range benchmarkData {
		b.Run(bd.name, func(b *testing.B) {
			// Serialize once
			data, err := jsonSerializer.Serialize(bd.data)
			if err != nil {
				b.Fatalf("Serialize failed: %v", err)
			}
			dataString := string(data)

			// Prepare result variable
			var result any
			switch bd.data.(type) {
			case string:
				result = ""
			case testStruct:
				result = testStruct{}
			default:
				result = bd.data
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				err := stringDeser.DeserializeString(dataString, &result)
				if err != nil {
					b.Fatalf("DeserializeString failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkJSONDeserializeBytes benchmarks JSON traditional byte-based deserialization
func BenchmarkJSONDeserializeBytes(b *testing.B) {
	jsonSerializer := serializer.NewJSONSerializer()

	for _, bd := range benchmarkData {
		b.Run(bd.name, func(b *testing.B) {
			// Serialize once
			data, err := jsonSerializer.Serialize(bd.data)
			if err != nil {
				b.Fatalf("Serialize failed: %v", err)
			}
			dataString := string(data)

			// Prepare result variable
			var result any
			switch bd.data.(type) {
			case string:
				result = ""
			case testStruct:
				result = testStruct{}
			default:
				result = bd.data
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Simulate string->[]byte conversion that happens in real usage
				dataBytes := []byte(dataString)
				err := jsonSerializer.Deserialize(dataBytes, &result)
				if err != nil {
					b.Fatalf("Deserialize failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkMsgPackDeserializeString benchmarks MsgPack StringDeserializer performance
func BenchmarkMsgPackDeserializeString(b *testing.B) {
	msgpackSerializer := serializer.NewMsgpackSerializer()
	stringDeser := msgpackSerializer.(serializer.StringDeserializer)

	for _, bd := range benchmarkData {
		b.Run(bd.name, func(b *testing.B) {
			// Serialize once
			data, err := msgpackSerializer.Serialize(bd.data)
			if err != nil {
				b.Fatalf("Serialize failed: %v", err)
			}
			dataString := string(data)

			// Prepare result variable
			var result any
			switch bd.data.(type) {
			case string:
				var v string
				result = &v
			case testStruct:
				var v testStruct
				result = &v
			default:
				result = &bd.data
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				err := stringDeser.DeserializeString(dataString, result)
				if err != nil {
					b.Fatalf("DeserializeString failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkMsgPackDeserializeBytes benchmarks MsgPack traditional byte-based deserialization
func BenchmarkMsgPackDeserializeBytes(b *testing.B) {
	msgpackSerializer := serializer.NewMsgpackSerializer()

	for _, bd := range benchmarkData {
		b.Run(bd.name, func(b *testing.B) {
			// Serialize once
			data, err := msgpackSerializer.Serialize(bd.data)
			if err != nil {
				b.Fatalf("Serialize failed: %v", err)
			}
			dataString := string(data)

			// Prepare result variable
			var result any
			switch bd.data.(type) {
			case string:
				var v string
				result = &v
			case testStruct:
				var v testStruct
				result = &v
			default:
				result = &bd.data
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Simulate string->[]byte conversion that happens in real usage
				dataBytes := []byte(dataString)
				err := msgpackSerializer.Deserialize(dataBytes, result)
				if err != nil {
					b.Fatalf("Deserialize failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkGobDeserializeString benchmarks Gob StringDeserializer performance
func BenchmarkGobDeserializeString(b *testing.B) {
	gobSerializer := serializer.NewGobSerializer()
	stringDeser := gobSerializer.(serializer.StringDeserializer)

	for _, bd := range benchmarkData {
		b.Run(bd.name, func(b *testing.B) {
			// Serialize once
			data, err := gobSerializer.Serialize(bd.data)
			if err != nil {
				b.Fatalf("Serialize failed: %v", err)
			}
			dataString := string(data)

			// Prepare result variable
			var result any
			switch bd.data.(type) {
			case string:
				var v string
				result = &v
			case testStruct:
				var v testStruct
				result = &v
			default:
				result = &bd.data
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				err := stringDeser.DeserializeString(dataString, result)
				if err != nil {
					b.Fatalf("DeserializeString failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkGobDeserializeBytes benchmarks Gob traditional byte-based deserialization
func BenchmarkGobDeserializeBytes(b *testing.B) {
	gobSerializer := serializer.NewGobSerializer()

	for _, bd := range benchmarkData {
		b.Run(bd.name, func(b *testing.B) {
			// Serialize once
			data, err := gobSerializer.Serialize(bd.data)
			if err != nil {
				b.Fatalf("Serialize failed: %v", err)
			}
			dataString := string(data)

			// Prepare result variable
			var result any
			switch bd.data.(type) {
			case string:
				var v string
				result = &v
			case testStruct:
				var v testStruct
				result = &v
			default:
				result = &bd.data
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Simulate string->[]byte conversion that happens in real usage
				dataBytes := []byte(dataString)
				err := gobSerializer.Deserialize(dataBytes, result)
				if err != nil {
					b.Fatalf("Deserialize failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkAllSerializersComparison provides side-by-side comparison
func BenchmarkAllSerializersComparison(b *testing.B) {
	testData := "This is a medium-sized test string for comparative benchmarking across different serialization formats."
	
	serializers := []struct {
		name       string
		serializer serializer.Serializer
	}{
		{"JSON", serializer.NewJSONSerializer()},
		{"MsgPack", serializer.NewMsgpackSerializer()},
		{"Gob", serializer.NewGobSerializer()},
	}

	for _, s := range serializers {
		b.Run(s.name+"_String", func(b *testing.B) {
			stringDeser := s.serializer.(serializer.StringDeserializer)
			data, _ := s.serializer.Serialize(testData)
			dataString := string(data)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var result string
				if s.name == "JSON" {
					err := stringDeser.DeserializeString(dataString, &result)
					if err != nil {
						b.Fatal(err)
					}
				} else {
					err := stringDeser.DeserializeString(dataString, &result)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})

		b.Run(s.name+"_Bytes", func(b *testing.B) {
			data, _ := s.serializer.Serialize(testData)
			dataString := string(data)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				var result string
				dataBytes := []byte(dataString)
				if s.name == "JSON" {
					err := s.serializer.Deserialize(dataBytes, &result)
					if err != nil {
						b.Fatal(err)
					}
				} else {
					err := s.serializer.Deserialize(dataBytes, &result)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}