package serializer

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

type testStruct struct {
	ID   int    `msgpack:"id"`
	Name string `msgpack:"name"`
	Data []byte `msgpack:"data"`
}

func TestPooledEncoder_BasicOperation(t *testing.T) {
	// Test basic encoder acquisition and release
	pe1 := getPooledEncoder()
	if pe1 == nil {
		t.Fatal("getPooledEncoder returned nil")
	}
	if pe1.enc == nil {
		t.Fatal("pooledEncoder.enc is nil")
	}
	if pe1.buf == nil {
		t.Fatal("pooledEncoder.buf is nil")
	}

	// Test encoding
	testValue := testStruct{ID: 1, Name: "test", Data: []byte("hello")}
	pe1.buf.Reset()
	pe1.enc.Reset(pe1.buf)
	
	err := pe1.enc.Encode(testValue)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	
	if pe1.buf.Len() == 0 {
		t.Fatal("Buffer is empty after encoding")
	}

	encoded1 := make([]byte, pe1.buf.Len())
	copy(encoded1, pe1.buf.Bytes())
	
	putPooledEncoder(pe1)

	// Test that we can get another encoder and it works
	pe2 := getPooledEncoder()
	if pe2 == nil {
		t.Fatal("second getPooledEncoder returned nil")
	}
	
	pe2.buf.Reset()
	pe2.enc.Reset(pe2.buf)
	
	err = pe2.enc.Encode(testValue)
	if err != nil {
		t.Fatalf("Second encode failed: %v", err)
	}
	
	encoded2 := make([]byte, pe2.buf.Len())
	copy(encoded2, pe2.buf.Bytes())
	
	putPooledEncoder(pe2)

	// Verify both encodings are identical
	if len(encoded1) != len(encoded2) {
		t.Fatalf("Encoded lengths differ: %d vs %d", len(encoded1), len(encoded2))
	}
	
	for i, b := range encoded1 {
		if b != encoded2[i] {
			t.Fatalf("Encoded bytes differ at position %d: %d vs %d", i, b, encoded2[i])
		}
	}
}

func TestPooledEncoder_ReusesEncoders(t *testing.T) {
	// Get an encoder and put it back
	pe1 := getPooledEncoder()
	originalPtr := pe1
	putPooledEncoder(pe1)
	
	// Get another encoder - it should be the same instance if pool is working
	pe2 := getPooledEncoder()
	
	// They should be the same pooledEncoder instance (pointer equality)
	if pe1 != pe2 {
		// This might not always be true due to pool implementation, but let's at least
		// verify we can encode with the reused encoder
		t.Logf("Different encoder instances (expected with some pool implementations)")
	}
	
	// Verify the reused encoder works correctly
	testValue := testStruct{ID: 42, Name: "reuse test", Data: []byte("reused")}
	pe2.buf.Reset()
	pe2.enc.Reset(pe2.buf)
	
	err := pe2.enc.Encode(testValue)
	if err != nil {
		t.Fatalf("Reused encoder failed to encode: %v", err)
	}
	
	if pe2.buf.Len() == 0 {
		t.Fatal("Reused encoder produced empty output")
	}
	
	putPooledEncoder(pe2)
	_ = originalPtr // avoid unused variable
}

func TestPooledEncoder_BufferCapacityLimit(t *testing.T) {
	pe := getPooledEncoder()
	
	// Create a large buffer that exceeds MAX_BUF_CAP
	largeData := make([]byte, MAX_BUF_CAP+1000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	
	largeStruct := testStruct{ID: 999, Name: "large", Data: largeData}
	
	pe.buf.Reset()
	pe.enc.Reset(pe.buf)
	
	err := pe.enc.Encode(largeStruct)
	if err != nil {
		t.Fatalf("Failed to encode large struct: %v", err)
	}
	
	// Buffer should now exceed MAX_BUF_CAP
	if pe.buf.Cap() <= MAX_BUF_CAP {
		t.Logf("Buffer capacity %d is not greater than MAX_BUF_CAP %d, test may not be effective", pe.buf.Cap(), MAX_BUF_CAP)
	}
	
	originalBuf := pe.buf
	
	// Return to pool - this should create a new buffer due to size limit
	putPooledEncoder(pe)
	
	// Get the encoder back and verify it has a fresh buffer (or at least capped capacity)
	pe2 := getPooledEncoder()
	
	// The buffer should either be a new instance or have been reset to a smaller capacity
	if pe2.buf == originalBuf && pe2.buf.Cap() > MAX_BUF_CAP {
		t.Errorf("Large buffer was not replaced: capacity %d > %d", pe2.buf.Cap(), MAX_BUF_CAP)
	}
	
	putPooledEncoder(pe2)
}

func TestPooledEncoder_ConcurrentStress(t *testing.T) {
	numGoroutines := 100
	numOperations := 1000
	
	var wg sync.WaitGroup
	
	// Test data of varying sizes
	testData := []testStruct{
		{ID: 1, Name: "small", Data: []byte("small")},
		{ID: 2, Name: "medium", Data: make([]byte, 1024)},
		{ID: 3, Name: "large", Data: make([]byte, 10*1024)},
		{ID: 4, Name: "huge", Data: make([]byte, MAX_BUF_CAP/2)},
	}
	
	// Fill test data with predictable content
	for i := range testData {
		for j := range testData[i].Data {
			testData[i].Data[j] = byte(j % 256)
		}
	}
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				// Get encoder from pool
				pe := getPooledEncoder()
				if pe == nil {
					t.Errorf("Goroutine %d: getPooledEncoder returned nil", goroutineID)
					return
				}
				
				// Pick test data based on operation index
				testValue := testData[j%len(testData)]
				testValue.ID = goroutineID*numOperations + j // Make each operation unique
				
				// Reset and encode
				pe.buf.Reset()
				pe.enc.Reset(pe.buf)
				
				err := pe.enc.Encode(testValue)
				if err != nil {
					t.Errorf("Goroutine %d, op %d: Encode failed: %v", goroutineID, j, err)
					putPooledEncoder(pe)
					return
				}
				
				if pe.buf.Len() == 0 {
					t.Errorf("Goroutine %d, op %d: Buffer empty after encode", goroutineID, j)
					putPooledEncoder(pe)
					return
				}
				
				// Verify we can read the encoded data (basic sanity check)
				encoded := pe.buf.Bytes()
				if len(encoded) < 4 { // MessagePack should produce at least a few bytes
					t.Errorf("Goroutine %d, op %d: Encoded data too short: %d bytes", goroutineID, j, len(encoded))
				}
				
				// Return encoder to pool
				putPooledEncoder(pe)
				
				// Occasionally yield to other goroutines
				if j%100 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}
	
	wg.Wait()
}

func TestPooledEncoder_BufferCapManagement(t *testing.T) {
	// This test verifies that large buffers are properly discarded to prevent memory bloat
	
	// Create data that will result in a buffer exceeding MAX_BUF_CAP
	largeData := make([]byte, MAX_BUF_CAP+10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	
	largeStruct := testStruct{ID: 999, Name: "memory_test", Data: largeData}
	
	// Track buffer instances to verify new ones are created
	var originalBuffers []*bytes.Buffer
	
	// Create several large encodings that should trigger buffer replacement
	for i := 0; i < 5; i++ {
		pe := getPooledEncoder()
		
		pe.buf.Reset()
		pe.enc.Reset(pe.buf)
		
		err := pe.enc.Encode(largeStruct)
		if err != nil {
			t.Fatalf("Failed to encode large struct in iteration %d: %v", i, err)
		}
		
		// Verify the buffer grew as expected
		if pe.buf.Cap() <= MAX_BUF_CAP {
			t.Logf("Iteration %d: Buffer capacity %d not greater than MAX_BUF_CAP %d", i, pe.buf.Cap(), MAX_BUF_CAP)
		}
		
		originalBuffers = append(originalBuffers, pe.buf)
		
		// Return to pool - should trigger buffer replacement due to size
		putPooledEncoder(pe)
	}
	
	// Now do some small encodings and verify they work with fresh buffers
	smallStruct := testStruct{ID: 1, Name: "small", Data: []byte("small")}
	
	for i := 0; i < 5; i++ {
		pe := getPooledEncoder()
		
		pe.buf.Reset()
		pe.enc.Reset(pe.buf)
		
		err := pe.enc.Encode(smallStruct)
		if err != nil {
			t.Fatalf("Failed to encode small struct in cleanup iteration %d: %v", i, err)
		}
		
		// The capacity should be reasonable for small data
		if pe.buf.Cap() > MAX_BUF_CAP {
			// Find if this buffer was one of our original large buffers
			wasOriginal := false
			for _, orig := range originalBuffers {
				if pe.buf == orig {
					wasOriginal = true
					break
				}
			}
			if wasOriginal {
				t.Errorf("Large buffer was reused instead of being replaced: capacity %d", pe.buf.Cap())
			}
		}
		
		putPooledEncoder(pe)
	}
}

func TestPooledDecoder_BasicOperation(t *testing.T) {
	// Create test data by encoding first
	testValue := testStruct{ID: 1, Name: "test", Data: []byte("hello")}
	
	pe := getPooledEncoder()
	pe.buf.Reset()
	pe.enc.Reset(pe.buf)
	err := pe.enc.Encode(testValue)
	if err != nil {
		t.Fatalf("Failed to encode test data: %v", err)
	}
	testData := make([]byte, pe.buf.Len())
	copy(testData, pe.buf.Bytes())
	putPooledEncoder(pe)
	
	pd1 := getPooledDecoder(testData)
	if pd1 == nil {
		t.Fatal("getPooledDecoder returned nil")
	}
	if pd1.dec == nil {
		t.Fatal("pooledDecoder.dec is nil")
	}
	if pd1.reader == nil {
		t.Fatal("pooledDecoder.reader is nil")
	}

	// Test decoding
	var decoded testStruct
	err = pd1.dec.Decode(&decoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	
	putPooledDecoder(pd1)

	// Test that we can get another decoder and it works
	pd2 := getPooledDecoder(testData)
	if pd2 == nil {
		t.Fatal("second getPooledDecoder returned nil")
	}
	
	var decoded2 testStruct
	err = pd2.dec.Decode(&decoded2)
	if err != nil {
		t.Fatalf("Second decode failed: %v", err)
	}
	
	putPooledDecoder(pd2)

	// Verify both decodings are identical and match the original
	if decoded.ID != testValue.ID || decoded.Name != testValue.Name {
		t.Fatalf("First decoded value incorrect: got %+v, want %+v", decoded, testValue)
	}
	if decoded2.ID != testValue.ID || decoded2.Name != testValue.Name {
		t.Fatalf("Second decoded value incorrect: got %+v, want %+v", decoded2, testValue)
	}
	if decoded.ID != decoded2.ID || decoded.Name != decoded2.Name {
		t.Fatalf("Decoded values differ: %+v vs %+v", decoded, decoded2)
	}
}

func TestPooledDecoder_ReusesDecoders(t *testing.T) {
	// First, encode some test data
	testValue := testStruct{ID: 42, Name: "reuse test", Data: []byte("reused")}
	
	pe := getPooledEncoder()
	pe.buf.Reset()
	pe.enc.Reset(pe.buf)
	err := pe.enc.Encode(testValue)
	if err != nil {
		t.Fatalf("Failed to encode test data: %v", err)
	}
	encodedData := make([]byte, pe.buf.Len())
	copy(encodedData, pe.buf.Bytes())
	putPooledEncoder(pe)

	// Get a decoder and put it back
	pd1 := getPooledDecoder(encodedData)
	originalPtr := pd1
	putPooledDecoder(pd1)
	
	// Get another decoder - it should be the same instance if pool is working
	pd2 := getPooledDecoder(encodedData)
	
	// They should be the same pooledDecoder instance (pointer equality)
	if pd1 != pd2 {
		// This might not always be true due to pool implementation, but let's at least
		// verify we can decode with the reused decoder
		t.Logf("Different decoder instances (expected with some pool implementations)")
	}
	
	// Verify the reused decoder works correctly
	var decoded testStruct
	err = pd2.dec.Decode(&decoded)
	if err != nil {
		t.Fatalf("Reused decoder failed to decode: %v", err)
	}
	
	if decoded.ID != testValue.ID || decoded.Name != testValue.Name {
		t.Fatalf("Decoded value incorrect: got %+v, want %+v", decoded, testValue)
	}
	
	putPooledDecoder(pd2)
	_ = originalPtr // avoid unused variable
}

func TestPooledDecoder_ConcurrentStress(t *testing.T) {
	numGoroutines := 100
	numOperations := 1000
	
	var wg sync.WaitGroup
	
	// Pre-encode test data of varying sizes
	testData := []testStruct{
		{ID: 1, Name: "small", Data: []byte("small")},
		{ID: 2, Name: "medium", Data: make([]byte, 1024)},
		{ID: 3, Name: "large", Data: make([]byte, 10*1024)},
	}
	
	// Fill test data with predictable content
	for i := range testData {
		for j := range testData[i].Data {
			testData[i].Data[j] = byte(j % 256)
		}
	}
	
	// Encode all test data
	encodedData := make([][]byte, len(testData))
	for i, testValue := range testData {
		pe := getPooledEncoder()
		pe.buf.Reset()
		pe.enc.Reset(pe.buf)
		err := pe.enc.Encode(testValue)
		if err != nil {
			t.Fatalf("Failed to encode test data %d: %v", i, err)
		}
		encodedData[i] = make([]byte, pe.buf.Len())
		copy(encodedData[i], pe.buf.Bytes())
		putPooledEncoder(pe)
	}
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				// Pick test data based on operation index
				dataIndex := j % len(encodedData)
				expectedValue := testData[dataIndex]
				
				// Get decoder from pool
				pd := getPooledDecoder(encodedData[dataIndex])
				if pd == nil {
					t.Errorf("Goroutine %d: getPooledDecoder returned nil", goroutineID)
					return
				}
				
				// Decode
				var decoded testStruct
				err := pd.dec.Decode(&decoded)
				if err != nil {
					t.Errorf("Goroutine %d, op %d: Decode failed: %v", goroutineID, j, err)
					putPooledDecoder(pd)
					return
				}
				
				// Verify correctness
				if decoded.ID != expectedValue.ID || decoded.Name != expectedValue.Name {
					t.Errorf("Goroutine %d, op %d: Decoded value incorrect: got %+v, want %+v", 
						goroutineID, j, decoded, expectedValue)
					putPooledDecoder(pd)
					return
				}
				
				// Verify data content
				if len(decoded.Data) != len(expectedValue.Data) {
					t.Errorf("Goroutine %d, op %d: Data length mismatch: got %d, want %d", 
						goroutineID, j, len(decoded.Data), len(expectedValue.Data))
					putPooledDecoder(pd)
					return
				}
				
				for k, b := range decoded.Data {
					if b != expectedValue.Data[k] {
						t.Errorf("Goroutine %d, op %d: Data content mismatch at pos %d: got %d, want %d", 
							goroutineID, j, k, b, expectedValue.Data[k])
						break
					}
				}
				
				// Return decoder to pool
				putPooledDecoder(pd)
				
				// Occasionally yield to other goroutines
				if j%100 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}
	
	wg.Wait()
}

func TestPooledEncoder_Decoder_RoundTrip(t *testing.T) {
	// Test full round-trip encode->decode with pooled infrastructure
	testCases := []testStruct{
		{ID: 1, Name: "simple", Data: []byte("test")},
		{ID: 42, Name: "with spaces", Data: []byte("hello world")},
		{ID: 999, Name: "empty data", Data: []byte{}},
		{ID: 0, Name: "", Data: []byte("empty name and zero ID")},
		{ID: -1, Name: "negative", Data: make([]byte, 1000)},
	}
	
	// Fill large data with pattern
	for i := range testCases[4].Data {
		testCases[4].Data[i] = byte(i % 256)
	}
	
	for i, original := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			// Encode with pooled encoder
			pe := getPooledEncoder()
			pe.buf.Reset()
			pe.enc.Reset(pe.buf)
			
			err := pe.enc.Encode(original)
			if err != nil {
				t.Fatalf("Encode failed: %v", err)
			}
			
			// Copy encoded data
			encodedData := make([]byte, pe.buf.Len())
			copy(encodedData, pe.buf.Bytes())
			putPooledEncoder(pe)
			
			// Decode with pooled decoder
			pd := getPooledDecoder(encodedData)
			var decoded testStruct
			err = pd.dec.Decode(&decoded)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			putPooledDecoder(pd)
			
			// Verify round-trip correctness
			if decoded.ID != original.ID {
				t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
			}
			if decoded.Name != original.Name {
				t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
			}
			if len(decoded.Data) != len(original.Data) {
				t.Errorf("Data length mismatch: got %d, want %d", len(decoded.Data), len(original.Data))
			}
			for j, b := range decoded.Data {
				if b != original.Data[j] {
					t.Errorf("Data[%d] mismatch: got %d, want %d", j, b, original.Data[j])
					break
				}
			}
		})
	}
}

func TestSerializeSafe_BasicOperation(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	testCases := []testStruct{
		{ID: 1, Name: "simple", Data: []byte("test")},
		{ID: 42, Name: "with spaces", Data: []byte("hello world")},
		{ID: 999, Name: "empty data", Data: []byte{}},
		{ID: 0, Name: "", Data: []byte("empty name and zero ID")},
		{ID: -1, Name: "negative", Data: make([]byte, 1000)},
	}
	
	// Fill large data with pattern
	for i := range testCases[4].Data {
		testCases[4].Data[i] = byte(i % 256)
	}
	
	for i, original := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			// Test SerializeSafe
			encoded, err := serializer.SerializeSafe(original)
			if err != nil {
				t.Fatalf("SerializeSafe failed: %v", err)
			}
			
			if len(encoded) == 0 {
				t.Fatal("SerializeSafe returned empty data")
			}
			
			// Test that we can deserialize it back
			var decoded testStruct
			err = msgpack.Unmarshal(encoded, &decoded)
			if err != nil {
				t.Fatalf("Failed to deserialize SerializeSafe output: %v", err)
			}
			
			// Verify round-trip correctness
			if decoded.ID != original.ID {
				t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
			}
			if decoded.Name != original.Name {
				t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
			}
			if len(decoded.Data) != len(original.Data) {
				t.Errorf("Data length mismatch: got %d, want %d", len(decoded.Data), len(original.Data))
			}
			for j, b := range decoded.Data {
				if b != original.Data[j] {
					t.Errorf("Data[%d] mismatch: got %d, want %d", j, b, original.Data[j])
					break
				}
			}
		})
	}
}

func TestSerializeSafe_vs_StandardSerialize(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	testValue := testStruct{ID: 42, Name: "comparison test", Data: []byte("compare")}
	
	// Serialize with SerializeSafe
	safeResult, err := serializer.SerializeSafe(testValue)
	if err != nil {
		t.Fatalf("SerializeSafe failed: %v", err)
	}
	
	// Serialize with standard Serialize (which now uses SerializeSafe internally)
	standardResult, err := serializer.Serialize(testValue)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	
	// Results should be identical
	if len(safeResult) != len(standardResult) {
		t.Fatalf("Result lengths differ: SerializeSafe=%d, Serialize=%d", len(safeResult), len(standardResult))
	}
	
	for i, b := range safeResult {
		if b != standardResult[i] {
			t.Fatalf("Results differ at position %d: SerializeSafe=%d, Serialize=%d", i, b, standardResult[i])
		}
	}
}

func TestSerializeSafe_NilValue(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	// Test nil value handling
	_, err := serializer.SerializeSafe(nil)
	if err == nil {
		t.Error("Expected error when serializing nil value")
	}
	
	expectedMsg := "cannot serialize nil value"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestSerializeSafe_ConcurrentStress(t *testing.T) {
	serializer := &MsgPackSerializer{}
	numGoroutines := 100
	numOperations := 1000
	
	var wg sync.WaitGroup
	
	testData := []testStruct{
		{ID: 1, Name: "small", Data: []byte("small")},
		{ID: 2, Name: "medium", Data: make([]byte, 1024)},
		{ID: 3, Name: "large", Data: make([]byte, 10*1024)},
		{ID: 4, Name: "huge", Data: make([]byte, MAX_BUF_CAP/2)},
	}
	
	// Fill test data with predictable content
	for i := range testData {
		for j := range testData[i].Data {
			testData[i].Data[j] = byte(j % 256)
		}
	}
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				// Pick test data based on operation index
				testValue := testData[j%len(testData)]
				testValue.ID = goroutineID*numOperations + j // Make each operation unique
				
				// Test SerializeSafe
				encoded, err := serializer.SerializeSafe(testValue)
				if err != nil {
					t.Errorf("Goroutine %d, op %d: SerializeSafe failed: %v", goroutineID, j, err)
					return
				}
				
				if len(encoded) == 0 {
					t.Errorf("Goroutine %d, op %d: SerializeSafe returned empty data", goroutineID, j)
					return
				}
				
				// Verify we can decode it
				var decoded testStruct
				err = msgpack.Unmarshal(encoded, &decoded)
				if err != nil {
					t.Errorf("Goroutine %d, op %d: Failed to decode SerializeSafe result: %v", goroutineID, j, err)
					return
				}
				
				// Basic sanity check
				if decoded.ID != testValue.ID {
					t.Errorf("Goroutine %d, op %d: ID mismatch: got %d, want %d", goroutineID, j, decoded.ID, testValue.ID)
					return
				}
				
				// Occasionally yield to other goroutines
				if j%100 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}
	
	wg.Wait()
}

func TestSerializeSafe_BufferOwnership(t *testing.T) {
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 123, Name: "ownership test", Data: []byte("ownership")}
	
	// Get first result
	result1, err := serializer.SerializeSafe(testValue)
	if err != nil {
		t.Fatalf("First SerializeSafe failed: %v", err)
	}
	
	// Modify the result to ensure it's owned by caller
	original := make([]byte, len(result1))
	copy(original, result1)
	
	// Corrupt the returned buffer
	for i := range result1 {
		result1[i] = 0xFF
	}
	
	// Get second result - should not be affected by first result corruption
	result2, err := serializer.SerializeSafe(testValue)
	if err != nil {
		t.Fatalf("Second SerializeSafe failed: %v", err)
	}
	
	// Second result should be identical to original first result
	if len(result2) != len(original) {
		t.Fatalf("Second result length differs: got %d, want %d", len(result2), len(original))
	}
	
	for i, b := range result2 {
		if b != original[i] {
			t.Fatalf("Second result differs at position %d: got %d, want %d", i, b, original[i])
		}
	}
	
	// Verify first result is still corrupted (proving ownership)
	corrupted := true
	for _, b := range result1 {
		if b != 0xFF {
			corrupted = false
			break
		}
	}
	if !corrupted {
		t.Error("First result was not properly corrupted, indicating shared buffer")
	}
}

func BenchmarkPooledDecoder_vs_Standard(b *testing.B) {
	// Create test data
	testValue := testStruct{ID: 42, Name: "benchmark test", Data: make([]byte, 1024)}
	for i := range testValue.Data {
		testValue.Data[i] = byte(i % 256)
	}
	
	// Encode the test data once
	pe := getPooledEncoder()
	pe.buf.Reset()
	pe.enc.Reset(pe.buf)
	err := pe.enc.Encode(testValue)
	if err != nil {
		b.Fatalf("Failed to encode test data: %v", err)
	}
	encodedData := make([]byte, pe.buf.Len())
	copy(encodedData, pe.buf.Bytes())
	putPooledEncoder(pe)
	
	b.Run("Standard_msgpack_Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var decoded testStruct
			err := msgpack.Unmarshal(encodedData, &decoded)
			if err != nil {
				b.Fatalf("Unmarshal failed: %v", err)
			}
		}
	})
	
	b.Run("Pooled_Decoder", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			pd := getPooledDecoder(encodedData)
			var decoded testStruct
			err := pd.dec.Decode(&decoded)
			if err != nil {
				b.Fatalf("Pooled decode failed: %v", err)
			}
			putPooledDecoder(pd)
		}
	})
}

func BenchmarkSerializeSafe_vs_Standard(b *testing.B) {
	// Create test data of various sizes
	testCases := []struct {
		name string
		data testStruct
	}{
		{
			name: "Small",
			data: testStruct{ID: 42, Name: "benchmark test", Data: make([]byte, 100)},
		},
		{
			name: "Medium", 
			data: testStruct{ID: 42, Name: "benchmark test", Data: make([]byte, 1024)},
		},
		{
			name: "Large",
			data: testStruct{ID: 42, Name: "benchmark test", Data: make([]byte, 10*1024)},
		},
	}
	
	// Fill test data with pattern
	for _, tc := range testCases {
		for i := range tc.data.Data {
			tc.data.Data[i] = byte(i % 256)
		}
	}
	
	serializer := &MsgPackSerializer{}
	
	for _, tc := range testCases {
		b.Run(tc.name+"_Standard_msgpack_Marshal", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := msgpack.Marshal(tc.data)
				if err != nil {
					b.Fatalf("msgpack.Marshal failed: %v", err)
				}
			}
		})
		
		b.Run(tc.name+"_SerializeSafe", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := serializer.SerializeSafe(tc.data)
				if err != nil {
					b.Fatalf("SerializeSafe failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkSerializeSafe_AllocationTest(b *testing.B) {
	// This benchmark specifically tests the allocation behavior
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 42, Name: "allocation test", Data: make([]byte, 1024)}
	
	// Fill with pattern
	for i := range testValue.Data {
		testValue.Data[i] = byte(i % 256)
	}
	
	b.Run("Standard_msgpack_Marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := msgpack.Marshal(testValue)
			if err != nil {
				b.Fatalf("msgpack.Marshal failed: %v", err)
			}
			// Prevent optimization that removes the allocation
			_ = result
		}
	})
	
	b.Run("SerializeSafe", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := serializer.SerializeSafe(testValue)
			if err != nil {
				b.Fatalf("SerializeSafe failed: %v", err)
			}
			// Prevent optimization that removes the allocation
			_ = result
		}
	})
}

func BenchmarkSerializeSafe_ConcurrentLoad(b *testing.B) {
	// Test performance under concurrent load to verify pool effectiveness
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 42, Name: "concurrent test", Data: make([]byte, 512)}
	
	// Fill with pattern
	for i := range testValue.Data {
		testValue.Data[i] = byte(i % 256)
	}
	
	b.Run("SerializeSafe_Concurrent", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(10) // Use 10x more goroutines than CPU cores
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := serializer.SerializeSafe(testValue)
				if err != nil {
					b.Errorf("SerializeSafe failed: %v", err)
					return
				}
			}
		})
	})
	
	b.Run("Standard_Marshal_Concurrent", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(10) // Use 10x more goroutines than CPU cores
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := msgpack.Marshal(testValue)
				if err != nil {
					b.Errorf("msgpack.Marshal failed: %v", err)
					return
				}
			}
		})
	})
}

func TestPooledBuf_BasicOperation(t *testing.T) {
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 42, Name: "pooled test", Data: []byte("pooled data")}
	
	// Test SerializePooled
	pb, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("SerializePooled failed: %v", err)
	}
	if pb == nil {
		t.Fatal("SerializePooled returned nil PooledBuf")
	}
	
	// Test Bytes() method
	bytes := pb.Bytes()
	if len(bytes) == 0 {
		t.Fatal("PooledBuf.Bytes() returned empty data")
	}
	
	// Test Len() method
	length := pb.Len()
	if length != len(bytes) {
		t.Fatalf("PooledBuf.Len() mismatch: got %d, want %d", length, len(bytes))
	}
	
	// Verify we can decode the data
	var decoded testStruct
	err = msgpack.Unmarshal(bytes, &decoded)
	if err != nil {
		t.Fatalf("Failed to decode PooledBuf bytes: %v", err)
	}
	
	// Verify correctness
	if decoded.ID != testValue.ID || decoded.Name != testValue.Name {
		t.Fatalf("Decoded value incorrect: got %+v, want %+v", decoded, testValue)
	}
	if len(decoded.Data) != len(testValue.Data) {
		t.Fatalf("Data length mismatch: got %d, want %d", len(decoded.Data), len(testValue.Data))
	}
	for i, b := range decoded.Data {
		if b != testValue.Data[i] {
			t.Fatalf("Data[%d] mismatch: got %d, want %d", i, b, testValue.Data[i])
			break
		}
	}
	
	// Test Release()
	pb.Release()
	
	// After Release(), methods should handle gracefully
	if pb.Bytes() != nil {
		t.Error("PooledBuf.Bytes() should return nil after Release()")
	}
	if pb.Len() != 0 {
		t.Error("PooledBuf.Len() should return 0 after Release()")
	}
	
	// Multiple Release() calls should be safe
	pb.Release() // Should not panic
}

func TestSerializePooled_vs_SerializeSafe(t *testing.T) {
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 123, Name: "comparison", Data: []byte("compare safe vs pooled")}
	
	// Serialize with SerializeSafe
	safeBytes, err := serializer.SerializeSafe(testValue)
	if err != nil {
		t.Fatalf("SerializeSafe failed: %v", err)
	}
	
	// Serialize with SerializePooled
	pb, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("SerializePooled failed: %v", err)
	}
	defer pb.Release()
	
	pooledBytes := pb.Bytes()
	
	// Results should be identical
	if len(safeBytes) != len(pooledBytes) {
		t.Fatalf("Result lengths differ: SerializeSafe=%d, SerializePooled=%d", len(safeBytes), len(pooledBytes))
	}
	
	for i, b := range safeBytes {
		if b != pooledBytes[i] {
			t.Fatalf("Results differ at position %d: SerializeSafe=%d, SerializePooled=%d", i, b, pooledBytes[i])
		}
	}
}

func TestPooledBuf_ZeroCopyBehavior(t *testing.T) {
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 999, Name: "zero copy test", Data: []byte("zero copy")}
	
	pb, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("SerializePooled failed: %v", err)
	}
	defer pb.Release()
	
	// Get bytes twice - should return the same underlying slice
	bytes1 := pb.Bytes()
	bytes2 := pb.Bytes()
	
	// Check if they point to the same underlying array (zero-copy)
	if &bytes1[0] != &bytes2[0] {
		t.Error("PooledBuf.Bytes() does not return the same underlying slice (not zero-copy)")
	}
	
	// Modify one slice and verify the other is affected (proving shared backing array)
	if len(bytes1) > 0 {
		original := bytes1[0]
		bytes1[0] = 0xFF
		
		if bytes2[0] != 0xFF {
			t.Error("Modifying one slice did not affect the other (not zero-copy)")
		}
		
		// Restore original value
		bytes1[0] = original
	}
}

func TestPooledBuf_LifecycleManagement(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	// Test that encoder is properly returned to pool
	testValue := testStruct{ID: 42, Name: "lifecycle", Data: make([]byte, 1024)}
	for i := range testValue.Data {
		testValue.Data[i] = byte(i % 256)
	}
	
	// Create and release a pooled buffer
	pb1, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("SerializePooled failed: %v", err)
	}
	
	// Store the buffer pointer to verify reuse
	originalBuf := pb1.pe.buf
	pb1.Release()
	
	// Create another pooled buffer - it might reuse the same buffer
	pb2, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("Second SerializePooled failed: %v", err)
	}
	defer pb2.Release()
	
	// While we can't guarantee the same buffer will be reused (due to pool implementation),
	// we can at least verify that the second operation works correctly
	bytes2 := pb2.Bytes()
	if len(bytes2) == 0 {
		t.Fatal("Second PooledBuf returned empty bytes")
	}
	
	// Decode to verify correctness
	var decoded testStruct
	err = msgpack.Unmarshal(bytes2, &decoded)
	if err != nil {
		t.Fatalf("Failed to decode second PooledBuf: %v", err)
	}
	
	if decoded.ID != testValue.ID {
		t.Fatalf("Second decode incorrect: got ID=%d, want %d", decoded.ID, testValue.ID)
	}
	
	_ = originalBuf // Prevent unused variable warning
}

func TestPooledBuf_LargeBufferHandling(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	// Create data that will exceed MAX_BUF_CAP when encoded
	largeData := make([]byte, MAX_BUF_CAP+1000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	
	largeValue := testStruct{ID: 999, Name: "large buffer test", Data: largeData}
	
	pb, err := serializer.SerializePooled(largeValue)
	if err != nil {
		t.Fatalf("SerializePooled with large data failed: %v", err)
	}
	
	// Verify the data is correct
	bytes := pb.Bytes()
	if len(bytes) == 0 {
		t.Fatal("SerializePooled returned empty bytes for large data")
	}
	
	// Verify we can decode it
	var decoded testStruct
	err = msgpack.Unmarshal(bytes, &decoded)
	if err != nil {
		t.Fatalf("Failed to decode large PooledBuf: %v", err)
	}
	
	if decoded.ID != largeValue.ID || len(decoded.Data) != len(largeValue.Data) {
		t.Fatalf("Large data decode incorrect: got ID=%d len=%d, want ID=%d len=%d", 
			decoded.ID, len(decoded.Data), largeValue.ID, len(largeValue.Data))
	}
	
	// Release - this should trigger buffer capacity management in putPooledEncoder
	pb.Release()
}

func TestCopyAndRelease_Helper(t *testing.T) {
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 555, Name: "copy test", Data: []byte("copy helper")}
	
	pb, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("SerializePooled failed: %v", err)
	}
	
	// Get original bytes for comparison
	originalBytes := make([]byte, pb.Len())
	copy(originalBytes, pb.Bytes())
	
	// Use CopyAndRelease helper
	copiedBytes := CopyAndRelease(pb)
	
	// Verify the copy is correct
	if len(copiedBytes) != len(originalBytes) {
		t.Fatalf("CopyAndRelease length mismatch: got %d, want %d", len(copiedBytes), len(originalBytes))
	}
	
	for i, b := range copiedBytes {
		if b != originalBytes[i] {
			t.Fatalf("CopyAndRelease data mismatch at %d: got %d, want %d", i, b, originalBytes[i])
		}
	}
	
	// Verify PooledBuf was released (should return nil/0 now)
	if pb.Bytes() != nil {
		t.Error("PooledBuf was not properly released by CopyAndRelease")
	}
	if pb.Len() != 0 {
		t.Error("PooledBuf length not 0 after CopyAndRelease")
	}
	
	// Verify the copied bytes can be decoded
	var decoded testStruct
	err = msgpack.Unmarshal(copiedBytes, &decoded)
	if err != nil {
		t.Fatalf("Failed to decode CopyAndRelease result: %v", err)
	}
	
	if decoded.ID != testValue.ID || decoded.Name != testValue.Name {
		t.Fatalf("CopyAndRelease decode incorrect: got %+v, want %+v", decoded, testValue)
	}
}

func TestCopyAndRelease_NilHandling(t *testing.T) {
	// Test nil PooledBuf
	result := CopyAndRelease(nil)
	if result != nil {
		t.Error("CopyAndRelease(nil) should return nil")
	}
	
	// Test PooledBuf with nil bytes (after Release)
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 1, Name: "test", Data: []byte("test")}
	
	pb, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("SerializePooled failed: %v", err)
	}
	
	// Release first, then try CopyAndRelease
	pb.Release()
	result = CopyAndRelease(pb)
	if result != nil {
		t.Error("CopyAndRelease on released PooledBuf should return nil")
	}
}

func TestSerializePooled_ErrorHandling(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	// Test nil value
	pb, err := serializer.SerializePooled(nil)
	if err == nil {
		t.Error("Expected error when serializing nil value")
	}
	if pb != nil {
		t.Error("PooledBuf should be nil on error")
	}
	
	expectedMsg := "cannot serialize nil value"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestSerializePooled_ConcurrentStress(t *testing.T) {
	serializer := &MsgPackSerializer{}
	numGoroutines := 50
	numOperations := 500
	
	var wg sync.WaitGroup
	
	testData := []testStruct{
		{ID: 1, Name: "small", Data: []byte("small")},
		{ID: 2, Name: "medium", Data: make([]byte, 1024)},
		{ID: 3, Name: "large", Data: make([]byte, 10*1024)},
	}
	
	// Fill test data with predictable content
	for i := range testData {
		for j := range testData[i].Data {
			testData[i].Data[j] = byte(j % 256)
		}
	}
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < numOperations; j++ {
				// Pick test data based on operation index
				testValue := testData[j%len(testData)]
				testValue.ID = goroutineID*numOperations + j // Make each operation unique
				
				// Test SerializePooled
				pb, err := serializer.SerializePooled(testValue)
				if err != nil {
					t.Errorf("Goroutine %d, op %d: SerializePooled failed: %v", goroutineID, j, err)
					return
				}
				
				if pb == nil {
					t.Errorf("Goroutine %d, op %d: SerializePooled returned nil", goroutineID, j)
					return
				}
				
				bytes := pb.Bytes()
				if len(bytes) == 0 {
					t.Errorf("Goroutine %d, op %d: PooledBuf returned empty bytes", goroutineID, j)
					pb.Release()
					return
				}
				
				// Verify we can decode it
				var decoded testStruct
				err = msgpack.Unmarshal(bytes, &decoded)
				if err != nil {
					t.Errorf("Goroutine %d, op %d: Failed to decode PooledBuf: %v", goroutineID, j, err)
					pb.Release()
					return
				}
				
				// Basic sanity check
				if decoded.ID != testValue.ID {
					t.Errorf("Goroutine %d, op %d: ID mismatch: got %d, want %d", goroutineID, j, decoded.ID, testValue.ID)
				}
				
				// Release the buffer
				pb.Release()
				
				// Occasionally yield to other goroutines
				if j%100 == 0 {
					runtime.Gosched()
				}
			}
		}(i)
	}
	
	wg.Wait()
}

func TestPooledBuf_PipelineSimulation(t *testing.T) {
	// Simulate the SetMany pipeline scenario mentioned in the spec
	serializer := &MsgPackSerializer{}
	
	// Create multiple values to "set" in a pipeline
	values := []testStruct{
		{ID: 1, Name: "key1", Data: []byte("value1")},
		{ID: 2, Name: "key2", Data: []byte("value2")},
		{ID: 3, Name: "key3", Data: []byte("value3")},
		{ID: 4, Name: "key4", Data: []byte("value4")},
		{ID: 5, Name: "key5", Data: []byte("value5")},
	}
	
	// Step 1: Serialize all values with SerializePooled (like building pipeline commands)
	var pooledBufs []*PooledBuf
	var pipelineData [][]byte
	
	for i, value := range values {
		pb, err := serializer.SerializePooled(value)
		if err != nil {
			t.Fatalf("SerializePooled failed for value %d: %v", i, err)
		}
		pooledBufs = append(pooledBufs, pb)
		
		// Simulate passing bytes to pipeline command (like pipe.SetEX)
		bytes := pb.Bytes()
		pipelineData = append(pipelineData, bytes)
	}
	
	// Step 2: Simulate pipeline execution - verify all data is still valid
	for i, data := range pipelineData {
		if len(data) == 0 {
			t.Fatalf("Pipeline data %d is empty", i)
		}
		
		// Verify we can still decode the data (simulating successful Redis write)
		var decoded testStruct
		err := msgpack.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to decode pipeline data %d: %v", i, err)
		}
		
		if decoded.ID != values[i].ID || decoded.Name != values[i].Name {
			t.Fatalf("Pipeline data %d incorrect: got %+v, want %+v", i, decoded, values[i])
		}
	}
	
	// Step 3: After pipeline exec completes, release all pooled buffers
	for i, pb := range pooledBufs {
		if pb == nil {
			t.Fatalf("PooledBuf %d is nil", i)
		}
		pb.Release()
	}
	
	// Step 4: Verify buffers are properly released (should return nil now)
	for i, pb := range pooledBufs {
		if pb.Bytes() != nil {
			t.Errorf("PooledBuf %d not properly released", i)
		}
		if pb.Len() != 0 {
			t.Errorf("PooledBuf %d length not 0 after release", i)
		}
	}
}

// Additional test structs for comprehensive testing
type simpleStruct struct {
	Value int `msgpack:"value"`
}

type complexStruct struct {
	ID       int64             `msgpack:"id"`
	Name     string            `msgpack:"name"`
	Tags     []string          `msgpack:"tags"`
	Metadata map[string]string `msgpack:"metadata"`
	Data     []byte            `msgpack:"data"`
	Score    float64           `msgpack:"score"`
	Active   bool              `msgpack:"active"`
}

type nestedStruct struct {
	Outer    string       `msgpack:"outer"`
	Inner    simpleStruct `msgpack:"inner"`
	Children []testStruct `msgpack:"children"`
}

func TestDeserialize_PooledDecoders_MultipleStructShapes(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	testCases := []struct {
		name  string
		value any
	}{
		{
			name:  "simple_struct",
			value: simpleStruct{Value: 42},
		},
		{
			name: "complex_struct",
			value: complexStruct{
				ID:       12345,
				Name:     "test complex",
				Tags:     []string{"tag1", "tag2", "tag3"},
				Metadata: map[string]string{"key1": "val1", "key2": "val2"},
				Data:     []byte("complex data content"),
				Score:    99.5,
				Active:   true,
			},
		},
		{
			name: "nested_struct",
			value: nestedStruct{
				Outer: "outer value",
				Inner: simpleStruct{Value: 123},
				Children: []testStruct{
					{ID: 1, Name: "child1", Data: []byte("data1")},
					{ID: 2, Name: "child2", Data: []byte("data2")},
				},
			},
		},
		{
			name: "original_test_struct",
			value: testStruct{
				ID:   999,
				Name: "pooled deserialize test",
				Data: make([]byte, 1000),
			},
		},
		{
			name:  "empty_struct",
			value: testStruct{},
		},
	}
	
	// Fill large data with pattern
	largeData := testCases[3].value.(testStruct)
	for i := range largeData.Data {
		largeData.Data[i] = byte(i % 256)
	}
	testCases[3].value = largeData
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First, serialize the value
			encoded, err := serializer.SerializeSafe(tc.value)
			if err != nil {
				t.Fatalf("SerializeSafe failed: %v", err)
			}
			
			// Test the updated Deserialize method (using pooled decoders)
			switch original := tc.value.(type) {
			case simpleStruct:
				var decoded simpleStruct
				err = serializer.Deserialize(encoded, &decoded)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				if decoded.Value != original.Value {
					t.Errorf("Simple struct mismatch: got %+v, want %+v", decoded, original)
				}
				
			case complexStruct:
				var decoded complexStruct
				err = serializer.Deserialize(encoded, &decoded)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				
				// Verify all fields
				if decoded.ID != original.ID {
					t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
				}
				if decoded.Name != original.Name {
					t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
				}
				if len(decoded.Tags) != len(original.Tags) {
					t.Errorf("Tags length mismatch: got %d, want %d", len(decoded.Tags), len(original.Tags))
				} else {
					for i, tag := range decoded.Tags {
						if tag != original.Tags[i] {
							t.Errorf("Tag[%d] mismatch: got %q, want %q", i, tag, original.Tags[i])
						}
					}
				}
				if len(decoded.Metadata) != len(original.Metadata) {
					t.Errorf("Metadata length mismatch: got %d, want %d", len(decoded.Metadata), len(original.Metadata))
				} else {
					for k, v := range original.Metadata {
						if decoded.Metadata[k] != v {
							t.Errorf("Metadata[%q] mismatch: got %q, want %q", k, decoded.Metadata[k], v)
						}
					}
				}
				if decoded.Score != original.Score {
					t.Errorf("Score mismatch: got %f, want %f", decoded.Score, original.Score)
				}
				if decoded.Active != original.Active {
					t.Errorf("Active mismatch: got %t, want %t", decoded.Active, original.Active)
				}
				
			case nestedStruct:
				var decoded nestedStruct
				err = serializer.Deserialize(encoded, &decoded)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				
				if decoded.Outer != original.Outer {
					t.Errorf("Outer mismatch: got %q, want %q", decoded.Outer, original.Outer)
				}
				if decoded.Inner.Value != original.Inner.Value {
					t.Errorf("Inner.Value mismatch: got %d, want %d", decoded.Inner.Value, original.Inner.Value)
				}
				if len(decoded.Children) != len(original.Children) {
					t.Errorf("Children length mismatch: got %d, want %d", len(decoded.Children), len(original.Children))
				}
				
			case testStruct:
				var decoded testStruct
				err = serializer.Deserialize(encoded, &decoded)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				
				if decoded.ID != original.ID {
					t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
				}
				if decoded.Name != original.Name {
					t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
				}
				if len(decoded.Data) != len(original.Data) {
					t.Errorf("Data length mismatch: got %d, want %d", len(decoded.Data), len(original.Data))
				}
				for i, b := range decoded.Data {
					if b != original.Data[i] {
						t.Errorf("Data[%d] mismatch: got %d, want %d", i, b, original.Data[i])
						break
					}
				}
			}
		})
	}
}

func TestDeserializeFromPooled_MultipleStructShapes(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	testCases := []struct {
		name  string
		value any
	}{
		{
			name:  "simple_struct",
			value: simpleStruct{Value: 999},
		},
		{
			name: "complex_struct",
			value: complexStruct{
				ID:       67890,
				Name:     "pooled complex test",
				Tags:     []string{"pooled", "test", "complex"},
				Metadata: map[string]string{"source": "pooled", "type": "test"},
				Data:     []byte("pooled complex data"),
				Score:    88.8,
				Active:   false,
			},
		},
		{
			name: "nested_struct",
			value: nestedStruct{
				Outer: "pooled outer",
				Inner: simpleStruct{Value: 456},
				Children: []testStruct{
					{ID: 10, Name: "pooled child1", Data: []byte("pooled data1")},
					{ID: 20, Name: "pooled child2", Data: []byte("pooled data2")},
					{ID: 30, Name: "pooled child3", Data: []byte("pooled data3")},
				},
			},
		},
		{
			name: "large_test_struct",
			value: testStruct{
				ID:   777,
				Name: "large pooled test",
				Data: make([]byte, 5000),
			},
		},
	}
	
	// Fill large data with pattern
	largeData := testCases[3].value.(testStruct)
	for i := range largeData.Data {
		largeData.Data[i] = byte((i * 7) % 256)
	}
	testCases[3].value = largeData
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize with SerializePooled
			pb, err := serializer.SerializePooled(tc.value)
			if err != nil {
				t.Fatalf("SerializePooled failed: %v", err)
			}
			defer pb.Release()
			
			// Test DeserializeFromPooled method
			switch original := tc.value.(type) {
			case simpleStruct:
				var decoded simpleStruct
				err = serializer.DeserializeFromPooled(pb, &decoded)
				if err != nil {
					t.Fatalf("DeserializeFromPooled failed: %v", err)
				}
				if decoded.Value != original.Value {
					t.Errorf("Simple struct mismatch: got %+v, want %+v", decoded, original)
				}
				
			case complexStruct:
				var decoded complexStruct
				err = serializer.DeserializeFromPooled(pb, &decoded)
				if err != nil {
					t.Fatalf("DeserializeFromPooled failed: %v", err)
				}
				
				// Verify complex structure fields
				if decoded.ID != original.ID || decoded.Name != original.Name {
					t.Errorf("Complex struct basic fields mismatch: got ID=%d Name=%q, want ID=%d Name=%q", 
						decoded.ID, decoded.Name, original.ID, original.Name)
				}
				if decoded.Score != original.Score || decoded.Active != original.Active {
					t.Errorf("Complex struct numeric fields mismatch: got Score=%f Active=%t, want Score=%f Active=%t",
						decoded.Score, decoded.Active, original.Score, original.Active)
				}
				
			case nestedStruct:
				var decoded nestedStruct
				err = serializer.DeserializeFromPooled(pb, &decoded)
				if err != nil {
					t.Fatalf("DeserializeFromPooled failed: %v", err)
				}
				
				if decoded.Outer != original.Outer {
					t.Errorf("Nested outer mismatch: got %q, want %q", decoded.Outer, original.Outer)
				}
				if decoded.Inner.Value != original.Inner.Value {
					t.Errorf("Nested inner mismatch: got %d, want %d", decoded.Inner.Value, original.Inner.Value)
				}
				if len(decoded.Children) != len(original.Children) {
					t.Errorf("Nested children length mismatch: got %d, want %d", len(decoded.Children), len(original.Children))
				}
				
			case testStruct:
				var decoded testStruct
				err = serializer.DeserializeFromPooled(pb, &decoded)
				if err != nil {
					t.Fatalf("DeserializeFromPooled failed: %v", err)
				}
				
				if decoded.ID != original.ID || decoded.Name != original.Name {
					t.Errorf("TestStruct mismatch: got ID=%d Name=%q, want ID=%d Name=%q", 
						decoded.ID, decoded.Name, original.ID, original.Name)
				}
				if len(decoded.Data) != len(original.Data) {
					t.Errorf("Data length mismatch: got %d, want %d", len(decoded.Data), len(original.Data))
				} else {
					// Check first and last few bytes to verify data integrity
					for i := 0; i < 10 && i < len(decoded.Data); i++ {
						if decoded.Data[i] != original.Data[i] {
							t.Errorf("Data[%d] mismatch: got %d, want %d", i, decoded.Data[i], original.Data[i])
						}
					}
					for i := len(decoded.Data) - 10; i < len(decoded.Data) && i >= 0; i++ {
						if decoded.Data[i] != original.Data[i] {
							t.Errorf("Data[%d] mismatch: got %d, want %d", i, decoded.Data[i], original.Data[i])
						}
					}
				}
			}
		})
	}
}

func TestDeserializeFromPooled_ErrorHandling(t *testing.T) {
	serializer := &MsgPackSerializer{}
	testValue := testStruct{ID: 42, Name: "error test", Data: []byte("test")}
	
	// Test nil PooledBuf
	var decoded testStruct
	err := serializer.DeserializeFromPooled(nil, &decoded)
	if err == nil {
		t.Error("Expected error when PooledBuf is nil")
	}
	if err.Error() != "PooledBuf is nil" {
		t.Errorf("Expected 'PooledBuf is nil', got %q", err.Error())
	}
	
	// Test nil output parameter
	pb, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("SerializePooled failed: %v", err)
	}
	defer pb.Release()
	
	err = serializer.DeserializeFromPooled(pb, nil)
	if err == nil {
		t.Error("Expected error when output parameter is nil")
	}
	if err.Error() != "output parameter is nil" {
		t.Errorf("Expected 'output parameter is nil', got %q", err.Error())
	}
	
	// Test released PooledBuf
	pb2, err := serializer.SerializePooled(testValue)
	if err != nil {
		t.Fatalf("SerializePooled failed: %v", err)
	}
	pb2.Release() // Release before using
	
	err = serializer.DeserializeFromPooled(pb2, &decoded)
	if err == nil {
		t.Error("Expected error when using released PooledBuf")
	}
	if err.Error() != "PooledBuf contains no data" {
		t.Errorf("Expected 'PooledBuf contains no data', got %q", err.Error())
	}
}

func TestDeserialize_ErrorHandling(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	// Test nil data
	var decoded testStruct
	err := serializer.Deserialize(nil, &decoded)
	if err == nil {
		t.Error("Expected error when data is nil")
	}
	if err.Error() != "data is nil" {
		t.Errorf("Expected 'data is nil', got %q", err.Error())
	}
	
	// Test nil output parameter
	validData, _ := serializer.SerializeSafe(testStruct{ID: 1, Name: "test", Data: []byte("data")})
	err = serializer.Deserialize(validData, nil)
	if err == nil {
		t.Error("Expected error when output parameter is nil")
	}
	if err.Error() != "output parameter is nil" {
		t.Errorf("Expected 'output parameter is nil', got %q", err.Error())
	}
	
	// Test invalid data
	invalidData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	err = serializer.Deserialize(invalidData, &decoded)
	if err == nil {
		t.Error("Expected error when deserializing invalid data")
	}
	// Don't check exact error message as it comes from msgpack library
}

func TestDeserialize_vs_DeserializeFromPooled_Consistency(t *testing.T) {
	serializer := &MsgPackSerializer{}
	
	testCases := []any{
		simpleStruct{Value: 123},
		complexStruct{
			ID:       999,
			Name:     "consistency test",
			Tags:     []string{"test1", "test2"},
			Metadata: map[string]string{"consistency": "check"},
			Data:     []byte("consistency data"),
			Score:    75.5,
			Active:   true,
		},
		testStruct{ID: 456, Name: "consistency", Data: make([]byte, 500)},
	}
	
	// Fill test data
	testData := testCases[2].(testStruct)
	for i := range testData.Data {
		testData.Data[i] = byte(i % 128)
	}
	testCases[2] = testData
	
	for i, original := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			// Encode once
			encoded, err := serializer.SerializeSafe(original)
			if err != nil {
				t.Fatalf("SerializeSafe failed: %v", err)
			}
			
			pb, err := serializer.SerializePooled(original)
			if err != nil {
				t.Fatalf("SerializePooled failed: %v", err)
			}
			defer pb.Release()
			
			// Decode with both methods and compare results
			switch original.(type) {
			case simpleStruct:
				var decoded1, decoded2 simpleStruct
				
				err = serializer.Deserialize(encoded, &decoded1)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				
				err = serializer.DeserializeFromPooled(pb, &decoded2)
				if err != nil {
					t.Fatalf("DeserializeFromPooled failed: %v", err)
				}
				
				if decoded1.Value != decoded2.Value {
					t.Errorf("Inconsistent results: Deserialize=%+v, DeserializeFromPooled=%+v", decoded1, decoded2)
				}
				
			case complexStruct:
				var decoded1, decoded2 complexStruct
				
				err = serializer.Deserialize(encoded, &decoded1)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				
				err = serializer.DeserializeFromPooled(pb, &decoded2)
				if err != nil {
					t.Fatalf("DeserializeFromPooled failed: %v", err)
				}
				
				if decoded1.ID != decoded2.ID || decoded1.Name != decoded2.Name {
					t.Errorf("Inconsistent basic fields: Deserialize ID=%d Name=%q, DeserializeFromPooled ID=%d Name=%q", 
						decoded1.ID, decoded1.Name, decoded2.ID, decoded2.Name)
				}
				
			case testStruct:
				var decoded1, decoded2 testStruct
				
				err = serializer.Deserialize(encoded, &decoded1)
				if err != nil {
					t.Fatalf("Deserialize failed: %v", err)
				}
				
				err = serializer.DeserializeFromPooled(pb, &decoded2)
				if err != nil {
					t.Fatalf("DeserializeFromPooled failed: %v", err)
				}
				
				if decoded1.ID != decoded2.ID || decoded1.Name != decoded2.Name {
					t.Errorf("Inconsistent results: Deserialize ID=%d Name=%q, DeserializeFromPooled ID=%d Name=%q", 
						decoded1.ID, decoded1.Name, decoded2.ID, decoded2.Name)
				}
				if len(decoded1.Data) != len(decoded2.Data) {
					t.Errorf("Data length inconsistent: Deserialize=%d, DeserializeFromPooled=%d", len(decoded1.Data), len(decoded2.Data))
				}
			}
		})
	}
}

// Comprehensive benchmarks to demonstrate allocation reduction from Step 5 implementation
func BenchmarkDeserialize_PooledVsStandard_AllocationReduction(b *testing.B) {
	serializer := &MsgPackSerializer{}
	
	// Create test data of various complexities to show allocation benefits
	testCases := []struct {
		name string
		data any
	}{
		{
			name: "Simple",
			data: simpleStruct{Value: 42},
		},
		{
			name: "Medium",
			data: testStruct{
				ID:   123,
				Name: "benchmark test",
				Data: make([]byte, 1024),
			},
		},
		{
			name: "Complex",
			data: complexStruct{
				ID:       999,
				Name:     "complex benchmark",
				Tags:     []string{"bench", "complex", "test", "allocation", "reduction"},
				Metadata: map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
				Data:     make([]byte, 2048),
				Score:    88.5,
				Active:   true,
			},
		},
		{
			name: "Large",
			data: testStruct{
				ID:   777,
				Name: "large benchmark test",
				Data: make([]byte, 10*1024),
			},
		},
	}
	
	// Fill data arrays with patterns
	for i, tc := range testCases {
		switch data := tc.data.(type) {
		case testStruct:
			for j := range data.Data {
				data.Data[j] = byte((i*1000 + j) % 256)
			}
			testCases[i].data = data
		case complexStruct:
			for j := range data.Data {
				data.Data[j] = byte((i*2000 + j*3) % 256)
			}
			testCases[i].data = data
		}
	}
	
	for _, tc := range testCases {
		// Encode the test data once for all benchmarks
		encoded, err := serializer.SerializeSafe(tc.data)
		if err != nil {
			b.Fatalf("Failed to encode test data for %s: %v", tc.name, err)
		}
		
		// Benchmark standard msgpack.Unmarshal
		b.Run(tc.name+"_Standard_Unmarshal", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				switch tc.data.(type) {
				case simpleStruct:
					var decoded simpleStruct
					err := msgpack.Unmarshal(encoded, &decoded)
					if err != nil {
						b.Fatalf("Standard unmarshal failed: %v", err)
					}
					_ = decoded // Prevent optimization
				case testStruct:
					var decoded testStruct
					err := msgpack.Unmarshal(encoded, &decoded)
					if err != nil {
						b.Fatalf("Standard unmarshal failed: %v", err)
					}
					_ = decoded // Prevent optimization
				case complexStruct:
					var decoded complexStruct
					err := msgpack.Unmarshal(encoded, &decoded)
					if err != nil {
						b.Fatalf("Standard unmarshal failed: %v", err)
					}
					_ = decoded // Prevent optimization
				}
			}
		})
		
		// Benchmark new pooled Deserialize method
		b.Run(tc.name+"_Pooled_Deserialize", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				switch tc.data.(type) {
				case simpleStruct:
					var decoded simpleStruct
					err := serializer.Deserialize(encoded, &decoded)
					if err != nil {
						b.Fatalf("Pooled Deserialize failed: %v", err)
					}
					_ = decoded // Prevent optimization
				case testStruct:
					var decoded testStruct
					err := serializer.Deserialize(encoded, &decoded)
					if err != nil {
						b.Fatalf("Pooled Deserialize failed: %v", err)
					}
					_ = decoded // Prevent optimization
				case complexStruct:
					var decoded complexStruct
					err := serializer.Deserialize(encoded, &decoded)
					if err != nil {
						b.Fatalf("Pooled Deserialize failed: %v", err)
					}
					_ = decoded // Prevent optimization
				}
			}
		})
	}
}

func BenchmarkDeserializeFromPooled_ZeroCopyBenefit(b *testing.B) {
	serializer := &MsgPackSerializer{}
	
	// Test scenarios that benefit most from zero-copy pooled deserialization
	testCases := []struct {
		name string
		data any
	}{
		{
			name: "Medium_ZeroCopy",
			data: testStruct{
				ID:   456,
				Name: "zero copy test",
				Data: make([]byte, 2048),
			},
		},
		{
			name: "Large_ZeroCopy",
			data: testStruct{
				ID:   999,
				Name: "large zero copy test",
				Data: make([]byte, 8*1024),
			},
		},
		{
			name: "Complex_ZeroCopy",
			data: complexStruct{
				ID:       12345,
				Name:     "complex zero copy",
				Tags:     []string{"zero", "copy", "pooled", "deserialize", "benchmark"},
				Metadata: map[string]string{"type": "benchmark", "method": "zero-copy", "size": "large"},
				Data:     make([]byte, 4*1024),
				Score:    95.5,
				Active:   true,
			},
		},
	}
	
	// Fill data with patterns
	for i, tc := range testCases {
		switch data := tc.data.(type) {
		case testStruct:
			for j := range data.Data {
				data.Data[j] = byte((i*500 + j*7) % 256)
			}
			testCases[i].data = data
		case complexStruct:
			for j := range data.Data {
				data.Data[j] = byte((i*750 + j*11) % 256)
			}
			testCases[i].data = data
		}
	}
	
	for _, tc := range testCases {
		// Create pooled buffer for zero-copy deserialization
		pb, err := serializer.SerializePooled(tc.data)
		if err != nil {
			b.Fatalf("Failed to create pooled buffer for %s: %v", tc.name, err)
		}
		
		// Also create standard encoded data for comparison
		encoded, err := serializer.SerializeSafe(tc.data)
		if err != nil {
			pb.Release()
			b.Fatalf("Failed to encode test data for %s: %v", tc.name, err)
		}
		
		// Benchmark standard deserialization from []byte
		b.Run(tc.name+"_Standard_FromBytes", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				switch tc.data.(type) {
				case testStruct:
					var decoded testStruct
					err := serializer.Deserialize(encoded, &decoded)
					if err != nil {
						b.Fatalf("Standard deserialize failed: %v", err)
					}
					_ = decoded // Prevent optimization
				case complexStruct:
					var decoded complexStruct
					err := serializer.Deserialize(encoded, &decoded)
					if err != nil {
						b.Fatalf("Standard deserialize failed: %v", err)
					}
					_ = decoded // Prevent optimization
				}
			}
		})
		
		// Benchmark zero-copy deserialization from PooledBuf
		b.Run(tc.name+"_ZeroCopy_FromPooledBuf", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				switch tc.data.(type) {
				case testStruct:
					var decoded testStruct
					err := serializer.DeserializeFromPooled(pb, &decoded)
					if err != nil {
						b.Fatalf("Zero-copy deserialize failed: %v", err)
					}
					_ = decoded // Prevent optimization
				case complexStruct:
					var decoded complexStruct
					err := serializer.DeserializeFromPooled(pb, &decoded)
					if err != nil {
						b.Fatalf("Zero-copy deserialize failed: %v", err)
					}
					_ = decoded // Prevent optimization
				}
			}
		})
		
		pb.Release() // Clean up after benchmarks
	}
}

func BenchmarkDeserialize_ConcurrentPerformance(b *testing.B) {
	serializer := &MsgPackSerializer{}
	
	// Test concurrent deserialization performance with pooled decoders
	testValue := complexStruct{
		ID:       555,
		Name:     "concurrent test",
		Tags:     []string{"concurrent", "benchmark", "pooled"},
		Metadata: map[string]string{"concurrency": "test", "pool": "decoder"},
		Data:     make([]byte, 1024),
		Score:    77.7,
		Active:   true,
	}
	
	// Fill with pattern
	for i := range testValue.Data {
		testValue.Data[i] = byte((i * 13) % 256)
	}
	
	encoded, err := serializer.SerializeSafe(testValue)
	if err != nil {
		b.Fatalf("Failed to encode test data: %v", err)
	}
	
	b.Run("Standard_Concurrent", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(8) // Use 8x more goroutines than CPU cores
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var decoded complexStruct
				err := msgpack.Unmarshal(encoded, &decoded)
				if err != nil {
					b.Errorf("Standard concurrent unmarshal failed: %v", err)
					return
				}
				_ = decoded // Prevent optimization
			}
		})
	})
	
	b.Run("Pooled_Concurrent", func(b *testing.B) {
		b.ReportAllocs()
		b.SetParallelism(8) // Use 8x more goroutines than CPU cores
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var decoded complexStruct
				err := serializer.Deserialize(encoded, &decoded)
				if err != nil {
					b.Errorf("Pooled concurrent deserialize failed: %v", err)
					return
				}
				_ = decoded // Prevent optimization
			}
		})
	})
}

func BenchmarkFullRoundTrip_PooledOptimization(b *testing.B) {
	// This benchmark demonstrates the full benefits of pooled serialization + deserialization
	serializer := &MsgPackSerializer{}
	
	testValue := complexStruct{
		ID:       888,
		Name:     "round trip test",
		Tags:     []string{"serialize", "deserialize", "round", "trip", "benchmark"},
		Metadata: map[string]string{"round": "trip", "full": "optimization", "pooled": "both"},
		Data:     make([]byte, 2048),
		Score:    92.3,
		Active:   false,
	}
	
	// Fill with pattern
	for i := range testValue.Data {
		testValue.Data[i] = byte((i * 17 + 42) % 256)
	}
	
	b.Run("Standard_RoundTrip", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Standard serialize + deserialize
			encoded, err := msgpack.Marshal(testValue)
			if err != nil {
				b.Fatalf("Standard marshal failed: %v", err)
			}
			
			var decoded complexStruct
			err = msgpack.Unmarshal(encoded, &decoded)
			if err != nil {
				b.Fatalf("Standard unmarshal failed: %v", err)
			}
			
			_ = decoded // Prevent optimization
		}
	})
	
	b.Run("PooledSafe_RoundTrip", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Pooled serialize safe + pooled deserialize
			encoded, err := serializer.SerializeSafe(testValue)
			if err != nil {
				b.Fatalf("SerializeSafe failed: %v", err)
			}
			
			var decoded complexStruct
			err = serializer.Deserialize(encoded, &decoded)
			if err != nil {
				b.Fatalf("Pooled Deserialize failed: %v", err)
			}
			
			_ = decoded // Prevent optimization
		}
	})
	
	b.Run("PooledZeroCopy_RoundTrip", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Zero-copy serialize + deserialize
			pb, err := serializer.SerializePooled(testValue)
			if err != nil {
				b.Fatalf("SerializePooled failed: %v", err)
			}
			
			var decoded complexStruct
			err = serializer.DeserializeFromPooled(pb, &decoded)
			if err != nil {
				pb.Release()
				b.Fatalf("DeserializeFromPooled failed: %v", err)
			}
			
			pb.Release()
			_ = decoded // Prevent optimization
		}
	})
}
