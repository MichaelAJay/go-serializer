package serializer

import (
	"sync"
	"testing"
	"time"
)

// TestBufferPoolBasicUsage tests basic buffer pool get/put operations
func TestBufferPoolBasicUsage(t *testing.T) {
	maxSize := 1024
	pool := newPooledBufferPool(maxSize)

	// Get a buffer
	buf1 := pool.Get()
	if buf1 == nil {
		t.Fatal("Expected buffer from pool, got nil")
	}

	// Buffer should be empty and ready for use
	if buf1.Len() != 0 {
		t.Errorf("Expected empty buffer, got length %d", buf1.Len())
	}

	// Write some data
	testData := "test data"
	buf1.WriteString(testData)

	if buf1.String() != testData {
		t.Errorf("Expected %q, got %q", testData, buf1.String())
	}

	// Put the buffer back
	pool.Put(buf1)

	// Get another buffer - should be the same one, but reset
	buf2 := pool.Get()
	if buf2 == nil {
		t.Fatal("Expected buffer from pool, got nil")
	}

	// Buffer should be reset (empty)
	if buf2.Len() != 0 {
		t.Errorf("Expected reset buffer to be empty, got length %d", buf2.Len())
	}

	// Should be the same underlying buffer (reused)
	if buf1 != buf2 {
		t.Log("Note: Buffers are different instances - this may be expected depending on pool implementation")
	}
}

// TestBufferPoolMaxSizeEnforcement tests that buffers exceeding maxSize are not returned to pool
func TestBufferPoolMaxSizeEnforcement(t *testing.T) {
	maxSize := 100 // Small max size for testing
	pool := newPooledBufferPool(maxSize)

	// Get a buffer and grow it beyond maxSize
	buf := pool.Get()
	largeData := make([]byte, maxSize+50) // Exceed max size
	for i := range largeData {
		largeData[i] = 'x'
	}
	buf.Write(largeData)

	if buf.Cap() <= maxSize {
		// Grow the buffer capacity explicitly if needed
		buf.Grow(maxSize + 100)
	}

	originalCap := buf.Cap()
	if originalCap <= maxSize {
		t.Skipf("Could not create buffer larger than maxSize (%d), got cap %d", maxSize, originalCap)
	}

	// Put the oversized buffer back
	pool.Put(buf)

	// Get a new buffer - should be a fresh one, not the oversized one
	newBuf := pool.Get()
	if newBuf.Cap() == originalCap {
		t.Errorf("Expected new buffer (oversized buffer should not be reused), but got same capacity %d", originalCap)
	}

	// The new buffer should be smaller than the oversized one
	if newBuf.Cap() >= originalCap {
		t.Logf("New buffer capacity %d >= original %d - this may indicate pool behavior has changed", newBuf.Cap(), originalCap)
	}
}

// TestBufferPoolConcurrentAccess tests concurrent access to the buffer pool
func TestBufferPoolConcurrentAccess(t *testing.T) {
	pool := newPooledBufferPool(1024)

	const numGoroutines = 50
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Track any errors from goroutines
	errChan := make(chan error, numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < operationsPerGoroutine; i++ {
				// Get buffer
				buf := pool.Get()
				if buf == nil {
					errChan <- &testError{"Got nil buffer from pool"}
					return
				}

				// Use buffer
				testData := "goroutine_" + string(rune('0'+goroutineID%10)) + "_op_" + string(rune('0'+i%10))
				buf.WriteString(testData)

				// Verify data
				if buf.String() != testData {
					errChan <- &testError{"Buffer data corruption: expected " + testData + ", got " + buf.String()}
					return
				}

				// Put buffer back
				pool.Put(buf)
			}
		}(g)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		t.Error(err)
	}
}

// TestBufferPoolMemoryLeaks tests that buffers are properly reset to prevent memory leaks
func TestBufferPoolMemoryLeaks(t *testing.T) {
	pool := newPooledBufferPool(1024)

	sensitiveData := "password123"

	// Get buffer and write sensitive data
	buf := pool.Get()
	buf.WriteString(sensitiveData)

	// Put buffer back
	pool.Put(buf)

	// Get a new buffer - should be reset
	newBuf := pool.Get()

	// Should not contain previous data
	bufContent := newBuf.String()
	if len(bufContent) > 0 {
		t.Errorf("Buffer not properly reset - contains data: %q", bufContent)
	}

	// Underlying bytes should also be clean
	bufBytes := newBuf.Bytes()
	for i, b := range bufBytes {
		if b != 0 {
			t.Errorf("Buffer bytes not properly reset at index %d: got %d", i, b)
			break
		}
	}

	// The string should not appear anywhere in the buffer's backing array
	bufStr := string(newBuf.Bytes()[:newBuf.Cap()])
	if len(bufStr) > 0 {
		// Check if any part contains the sensitive data
		for i := 0; i <= len(bufStr)-len(sensitiveData); i++ {
			if bufStr[i:i+len(sensitiveData)] == sensitiveData {
				t.Error("Sensitive data found in reset buffer - potential memory leak")
				break
			}
		}
	}
}

// TestBufferPoolDisabled tests behavior when maxBufferSize <= 0 (no size limit)
func TestBufferPoolDisabled(t *testing.T) {
	pool := newPooledBufferPool(0) // Disabled - no size limit

	// Get buffer and make it very large
	buf := pool.Get()
	largeData := make([]byte, 100*1024) // 100KB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	buf.Write(largeData)

	originalCap := buf.Cap()

	// Put the large buffer back - should be accepted since no size limit
	pool.Put(buf)

	// Get a new buffer - might be the same large one
	newBuf := pool.Get()

	// Since there's no size limit, the large buffer should be reusable
	// (though this depends on sync.Pool's internal behavior)
	if newBuf.Cap() < originalCap {
		t.Logf("Buffer capacity reduced from %d to %d - this may be due to sync.Pool's internal cleanup", originalCap, newBuf.Cap())
	}
}

// TestBufferPoolDifferentSizes tests pool behavior with various buffer sizes
func TestBufferPoolDifferentSizes(t *testing.T) {
	testCases := []int{512, 1024, 4096, 16384}

	for _, maxSize := range testCases {
		t.Run(string(rune('0'+maxSize/1000)), func(t *testing.T) {
			pool := newPooledBufferPool(maxSize)

			// Test with buffer smaller than max
			buf1 := pool.Get()
			smallData := make([]byte, maxSize/2)
			buf1.Write(smallData)
			pool.Put(buf1)

			// Test with buffer larger than max
			buf2 := pool.Get()
			largeData := make([]byte, maxSize+1)
			buf2.Write(largeData)
			
			if buf2.Cap() > maxSize {
				// This buffer should not be returned to pool
				pool.Put(buf2)

				// Next buffer should be different
				buf3 := pool.Get()
				if buf3.Cap() == buf2.Cap() {
					t.Logf("Large buffer may have been reused - pool behavior may differ from expected")
				}
			}
		})
	}
}

// TestBufferPoolGrowth tests buffer growth behavior
func TestBufferPoolGrowth(t *testing.T) {
	pool := newPooledBufferPool(8192)

	buf := pool.Get()
	initialCap := buf.Cap()

	// Write data that will cause buffer to grow
	for i := 0; i < 1000; i++ {
		buf.WriteString("This is a test string that will cause buffer growth. ")
	}

	finalCap := buf.Cap()
	if finalCap <= initialCap {
		t.Logf("Buffer capacity did not grow as expected: initial=%d, final=%d", initialCap, finalCap)
	}

	// Put buffer back
	pool.Put(buf)

	// Get new buffer
	newBuf := pool.Get()
	
	// Should be reset but may retain capacity
	if newBuf.Len() != 0 {
		t.Error("Buffer not properly reset after growth")
	}
}

// TestBufferPoolRealWorldUsage tests buffer pool with realistic JSON serialization workload
func TestBufferPoolRealWorldUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-world usage test in short mode")
	}

	serializer := NewJSONSerializer(1024).(*JSONSerializer)

	// Create realistic test data
	testData := map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": 1, "name": "Alice", "email": "alice@example.com", "active": true},
			{"id": 2, "name": "Bob", "email": "bob@example.com", "active": false},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com", "active": true},
		},
		"metadata": map[string]interface{}{
			"total":     3,
			"timestamp": "2024-01-01T00:00:00Z",
			"version":   "1.0.0",
		},
	}

	// Perform many serialization operations
	const operations = 1000
	for i := 0; i < operations; i++ {
		data, err := serializer.Serialize(testData)
		if err != nil {
			t.Fatalf("Serialization failed at operation %d: %v", i, err)
		}

		if len(data) == 0 {
			t.Fatalf("Empty serialization result at operation %d", i)
		}

		// Occasionally verify deserialization works
		if i%100 == 0 {
			var result map[string]interface{}
			if err := serializer.Deserialize(data, &result); err != nil {
				t.Fatalf("Deserialization failed at operation %d: %v", i, err)
			}
		}
	}
}

// TestBufferPoolStress tests buffer pool under stress conditions
func TestBufferPoolStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	pool := newPooledBufferPool(4096)

	const numGoroutines = 20
	const duration = 2 * time.Second

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	stopChan := make(chan struct{})

	// Stop after duration
	go func() {
		time.Sleep(duration)
		close(stopChan)
	}()

	for g := 0; g < numGoroutines; g++ {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-stopChan:
					return
				default:
					buf := pool.Get()
					buf.WriteString("stress test data")
					buf.WriteString(" with more content")
					pool.Put(buf)
				}
			}
		}()
	}

	wg.Wait()
}

