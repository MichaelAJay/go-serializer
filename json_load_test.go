package serializer

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestJSONConcurrentSerialization tests concurrent serialization operations
func TestJSONConcurrentSerialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent serialization test in short mode")
	}

	s := NewJSONSerializer(8192)

	const numGoroutines = 50
	const operationsPerGoroutine = 200

	// Test data of different sizes and complexities
	testData := []interface{}{
		map[string]interface{}{"simple": "data", "number": 42},
		generateComplexObject(100),
		generateArrayData(50),
		generateNestedData(5),
	}

	var wg sync.WaitGroup
	var errorCount int64
	var successCount int64

	start := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(int64(goroutineID)))

			for i := 0; i < operationsPerGoroutine; i++ {
				// Pick random test data
				data := testData[rng.Intn(len(testData))]

				// Add goroutine-specific data to avoid interference
				testObj := map[string]interface{}{
					"goroutine":  goroutineID,
					"operation":  i,
					"timestamp":  time.Now().UnixNano(),
					"data":       data,
				}

				// Serialize
				serialized, err := s.Serialize(testObj)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("Goroutine %d, operation %d: serialize error: %v", goroutineID, i, err)
					continue
				}

				if len(serialized) == 0 {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("Goroutine %d, operation %d: empty serialization result", goroutineID, i)
					continue
				}

				// Deserialize to verify correctness
				var result map[string]interface{}
				if err := s.Deserialize(serialized, &result); err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("Goroutine %d, operation %d: deserialize error: %v", goroutineID, i, err)
					continue
				}

				// Verify key fields
				if result["goroutine"] != float64(goroutineID) {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("Goroutine %d, operation %d: goroutine ID mismatch", goroutineID, i)
					continue
				}

				if result["operation"] != float64(i) {
					atomic.AddInt64(&errorCount, 1)
					t.Errorf("Goroutine %d, operation %d: operation ID mismatch", goroutineID, i)
					continue
				}

				atomic.AddInt64(&successCount, 1)
			}
		}(g)
	}

	wg.Wait()

	elapsed := time.Since(start)
	totalOperations := numGoroutines * operationsPerGoroutine

	t.Logf("Concurrent serialization test completed:")
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Successful operations: %d", successCount)
	t.Logf("  Failed operations: %d", errorCount)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Operations per second: %.2f", float64(totalOperations)/elapsed.Seconds())

	if errorCount > 0 {
		t.Errorf("Had %d errors out of %d operations", errorCount, totalOperations)
	}

	if successCount != int64(totalOperations) {
		t.Errorf("Expected %d successful operations, got %d", totalOperations, successCount)
	}
}

// TestJSONHighThroughput tests high-throughput serialization
func TestJSONHighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high throughput test in short mode")
	}

	s := NewJSONSerializer(16 * 1024)

	// Create test data of various sizes
	smallData := generateComplexObject(10)
	mediumData := generateComplexObject(100)
	largeData := generateComplexObject(1000)

	testCases := []struct {
		name string
		data interface{}
		targetOpsPerSec int
	}{
		{"Small", smallData, 10000},
		{"Medium", mediumData, 5000},
		{"Large", largeData, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			const duration = 5 * time.Second
			const warmupOps = 100

			// Warmup
			for i := 0; i < warmupOps; i++ {
				s.Serialize(tc.data)
			}

			// Measure throughput
			var operations int64
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), duration)
			defer cancel()

			// Run operations until timeout
			for {
				select {
				case <-ctx.Done():
					elapsed := time.Since(start)
					opsPerSec := float64(operations) / elapsed.Seconds()

					t.Logf("%s throughput: %.2f ops/sec (target: %d)", tc.name, opsPerSec, tc.targetOpsPerSec)

					if opsPerSec < float64(tc.targetOpsPerSec)*0.8 { // Allow 20% tolerance
						t.Logf("Warning: Throughput below 80%% of target for %s", tc.name)
					}

					return
				default:
					_, err := s.Serialize(tc.data)
					if err != nil {
						t.Fatalf("Serialization failed: %v", err)
					}
					atomic.AddInt64(&operations, 1)
				}
			}
		})
	}
}

// TestJSONMemoryStability tests memory usage stability under load
func TestJSONMemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory stability test in short mode")
	}

	s := NewJSONSerializer(32 * 1024)

	// Force garbage collection before test
	runtime.GC()
	runtime.GC() // Call twice to ensure clean state

	var initialMemStats runtime.MemStats
	runtime.ReadMemStats(&initialMemStats)

	const iterations = 10000
	testData := generateComplexObject(500)

	t.Logf("Starting memory stability test with %d iterations", iterations)
	t.Logf("Initial heap size: %d bytes", initialMemStats.HeapAlloc)

	start := time.Now()

	// Run many serialization operations
	for i := 0; i < iterations; i++ {
		// Add iteration-specific data to prevent optimization
		data := map[string]interface{}{
			"iteration": i,
			"timestamp": time.Now().UnixNano(),
			"data":      testData,
		}

		serialized, err := s.Serialize(data)
		if err != nil {
			t.Fatalf("Iteration %d: serialize failed: %v", i, err)
		}

		// Also deserialize to stress both paths
		var result map[string]interface{}
		if err := s.Deserialize(serialized, &result); err != nil {
			t.Fatalf("Iteration %d: deserialize failed: %v", i, err)
		}

		// Periodic garbage collection to test stability
		if i%1000 == 0 {
			runtime.GC()
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			t.Logf("Iteration %d: heap size: %d bytes", i, memStats.HeapAlloc)
		}
	}

	elapsed := time.Since(start)

	// Final memory check
	runtime.GC()
	runtime.GC()
	var finalMemStats runtime.MemStats
	runtime.ReadMemStats(&finalMemStats)

	memoryGrowth := int64(finalMemStats.HeapAlloc) - int64(initialMemStats.HeapAlloc)
	
	t.Logf("Memory stability test completed:")
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Operations per second: %.2f", float64(iterations)/elapsed.Seconds())
	t.Logf("  Initial heap: %d bytes", initialMemStats.HeapAlloc)
	t.Logf("  Final heap: %d bytes", finalMemStats.HeapAlloc)
	t.Logf("  Memory growth: %d bytes", memoryGrowth)
	t.Logf("  Growth per operation: %.2f bytes", float64(memoryGrowth)/float64(iterations))

	// Memory growth should be reasonable (allow up to 10MB growth)
	const maxMemoryGrowth = 10 * 1024 * 1024
	if memoryGrowth > maxMemoryGrowth {
		t.Errorf("Memory growth too large: %d bytes (limit: %d)", memoryGrowth, maxMemoryGrowth)
	}
}

// TestJSONWebAPIResponses tests realistic web API response scenarios
func TestJSONWebAPIResponses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping web API response test in short mode")
	}

	s := NewJSONSerializer(64 * 1024)

	// Simulate various API response types
	testCases := []struct {
		name     string
		generator func() interface{}
		count     int
	}{
		{
			name: "UserProfiles",
			generator: func() interface{} {
				return generateUserProfile()
			},
			count: 1000,
		},
		{
			name: "ProductCatalogs",
			generator: func() interface{} {
				return generateProductCatalog(50)
			},
			count: 100,
		},
		{
			name: "LogEntries",
			generator: func() interface{} {
				return generateLogEntry()
			},
			count: 5000,
		},
		{
			name: "APIErrors",
			generator: func() interface{} {
				return generateAPIError()
			},
			count: 500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			var totalSize int64

			for i := 0; i < tc.count; i++ {
				data := tc.generator()

				serialized, err := s.Serialize(data)
				if err != nil {
					t.Fatalf("Failed to serialize %s item %d: %v", tc.name, i, err)
				}

				totalSize += int64(len(serialized))

				// Verify deserialization works
				var result interface{}
				if err := s.Deserialize(serialized, &result); err != nil {
					t.Fatalf("Failed to deserialize %s item %d: %v", tc.name, i, err)
				}
			}

			elapsed := time.Since(start)
			avgSize := totalSize / int64(tc.count)
			
			t.Logf("%s test completed:", tc.name)
			t.Logf("  Items processed: %d", tc.count)
			t.Logf("  Duration: %v", elapsed)
			t.Logf("  Items per second: %.2f", float64(tc.count)/elapsed.Seconds())
			t.Logf("  Total size: %d bytes", totalSize)
			t.Logf("  Average size: %d bytes", avgSize)
		})
	}
}

// TestJSONConfigFiles tests configuration file serialization patterns
func TestJSONConfigFiles(t *testing.T) {
	s := NewJSONSerializer(8192)

	configTypes := []struct {
		name   string
		config interface{}
	}{
		{
			name: "DatabaseConfig",
			config: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"database": "myapp",
				"username": "user",
				"password": "secret",
				"sslmode":  "require",
				"pool": map[string]interface{}{
					"max_connections": 20,
					"idle_timeout":    "5m",
				},
			},
		},
		{
			name: "ServerConfig",
			config: map[string]interface{}{
				"port":         8080,
				"host":         "0.0.0.0",
				"read_timeout": "30s",
				"write_timeout": "30s",
				"tls": map[string]interface{}{
					"enabled":   true,
					"cert_file": "/etc/ssl/cert.pem",
					"key_file":  "/etc/ssl/key.pem",
				},
				"cors": map[string]interface{}{
					"allowed_origins": []string{"*"},
					"allowed_methods": []string{"GET", "POST", "PUT", "DELETE"},
				},
			},
		},
		{
			name: "LoggingConfig",
			config: map[string]interface{}{
				"level": "info",
				"format": "json",
				"outputs": []interface{}{
					map[string]interface{}{
						"type": "file",
						"path": "/var/log/app.log",
						"rotation": map[string]interface{}{
							"max_size":    "100MB",
							"max_backups": 5,
							"max_age":     "30d",
						},
					},
					map[string]interface{}{
						"type": "stdout",
						"format": "text",
					},
				},
			},
		},
	}

	for _, tc := range configTypes {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize config
			serialized, err := s.Serialize(tc.config)
			if err != nil {
				t.Fatalf("Failed to serialize %s: %v", tc.name, err)
			}

			// Verify it's valid JSON
			if len(serialized) == 0 {
				t.Error("Serialized config is empty")
			}

			// Deserialize and verify structure is maintained
			var result map[string]interface{}
			if err := s.Deserialize(serialized, &result); err != nil {
				t.Fatalf("Failed to deserialize %s: %v", tc.name, err)
			}

			// Basic structure validation
			originalMap := tc.config.(map[string]interface{})
			if len(result) != len(originalMap) {
				t.Errorf("Config structure changed: original had %d keys, result has %d", len(originalMap), len(result))
			}

			// Check that all top-level keys exist
			for key := range originalMap {
				if result[key] == nil {
					t.Errorf("Missing key in deserialized config: %s", key)
				}
			}
		})
	}
}

// TestJSONLogMessages tests log message serialization patterns
func TestJSONLogMessages(t *testing.T) {
	s := NewJSONSerializer(2048)

	const numLogMessages = 10000
	const numGoroutines = 10

	var wg sync.WaitGroup
	var errorCount int64

	start := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(int64(goroutineID)))

			for i := 0; i < numLogMessages/numGoroutines; i++ {
				logMessage := generateLogMessage(rng, goroutineID, i)

				serialized, err := s.Serialize(logMessage)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				// Verify deserialization
				var result map[string]interface{}
				if err := s.Deserialize(serialized, &result); err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				// Basic validation
				if result["level"] == nil || result["message"] == nil || result["timestamp"] == nil {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}(g)
	}

	wg.Wait()

	elapsed := time.Since(start)
	
	t.Logf("Log message serialization test completed:")
	t.Logf("  Messages processed: %d", numLogMessages)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Messages per second: %.2f", float64(numLogMessages)/elapsed.Seconds())
	t.Logf("  Error count: %d", errorCount)

	if errorCount > 0 {
		t.Errorf("Had %d errors out of %d log messages", errorCount, numLogMessages)
	}
}

// Helper functions for generating test data
func generateComplexObject(size int) map[string]interface{} {
	obj := make(map[string]interface{})
	
	for i := 0; i < size; i++ {
		key := "field_" + strconv.Itoa(i)
		switch i % 5 {
		case 0:
			obj[key] = "string_value_" + strconv.Itoa(i)
		case 1:
			obj[key] = i
		case 2:
			obj[key] = float64(i) * 3.14159
		case 3:
			obj[key] = i%2 == 0
		case 4:
			obj[key] = []string{"item1", "item2", "item3"}
		}
	}
	
	return obj
}

func generateArrayData(size int) []interface{} {
	arr := make([]interface{}, size)
	
	for i := 0; i < size; i++ {
		arr[i] = map[string]interface{}{
			"id":    i,
			"name":  "Item " + strconv.Itoa(i),
			"value": float64(i) * 1.5,
		}
	}
	
	return arr
}

func generateNestedData(depth int) interface{} {
	if depth <= 0 {
		return map[string]interface{}{
			"leaf": true,
			"value": "bottom_level",
		}
	}
	
	return map[string]interface{}{
		"level": depth,
		"child": generateNestedData(depth - 1),
		"data":  "level_" + strconv.Itoa(depth),
	}
}

func generateUserProfile() map[string]interface{} {
	return map[string]interface{}{
		"id":       rand.Intn(1000000),
		"username": "user_" + strconv.Itoa(rand.Intn(10000)),
		"email":    "user" + strconv.Itoa(rand.Intn(10000)) + "@example.com",
		"profile": map[string]interface{}{
			"firstName": "User",
			"lastName":  "Name",
			"age":       20 + rand.Intn(60),
			"location":  "City, Country",
		},
		"preferences": map[string]interface{}{
			"theme":         []string{"light", "dark"}[rand.Intn(2)],
			"notifications": rand.Intn(2) == 0,
			"language":      "en",
		},
		"timestamps": map[string]interface{}{
			"created": time.Now().Add(-time.Duration(rand.Intn(365)) * 24 * time.Hour).Format(time.RFC3339),
			"updated": time.Now().Format(time.RFC3339),
		},
	}
}

func generateProductCatalog(numProducts int) map[string]interface{} {
	products := make([]interface{}, numProducts)
	
	for i := 0; i < numProducts; i++ {
		products[i] = map[string]interface{}{
			"id":          i + 1,
			"name":        "Product " + strconv.Itoa(i+1),
			"description": "This is a description for product " + strconv.Itoa(i+1),
			"price":       float64(rand.Intn(10000)) / 100.0,
			"category":    "Category " + strconv.Itoa(rand.Intn(10)+1),
			"in_stock":    rand.Intn(2) == 0,
			"tags":        []string{"tag1", "tag2", "tag3"},
		}
	}
	
	return map[string]interface{}{
		"products": products,
		"total":    numProducts,
		"metadata": map[string]interface{}{
			"generated": time.Now().Format(time.RFC3339),
			"version":   "1.0",
		},
	}
}

func generateLogEntry() map[string]interface{} {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	messages := []string{
		"User logged in",
		"Database query executed",
		"Cache miss occurred",
		"Request processed successfully",
		"Configuration updated",
	}
	
	return map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"level":     levels[rand.Intn(len(levels))],
		"message":   messages[rand.Intn(len(messages))],
		"logger":    "app.service." + strconv.Itoa(rand.Intn(10)),
		"thread":    "thread-" + strconv.Itoa(rand.Intn(20)),
		"context": map[string]interface{}{
			"user_id":    rand.Intn(1000),
			"request_id": fmt.Sprintf("req-%d-%d", time.Now().Unix(), rand.Intn(1000)),
			"duration":   rand.Intn(1000),
		},
	}
}

func generateAPIError() map[string]interface{} {
	errorCodes := []string{"VALIDATION_ERROR", "NOT_FOUND", "UNAUTHORIZED", "INTERNAL_ERROR"}
	
	return map[string]interface{}{
		"error": map[string]interface{}{
			"code":    errorCodes[rand.Intn(len(errorCodes))],
			"message": "An error occurred while processing the request",
			"details": map[string]interface{}{
				"field":     "some_field",
				"value":     "invalid_value",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
		"request_id": fmt.Sprintf("req-%d-%d", time.Now().Unix(), rand.Intn(1000)),
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}

func generateLogMessage(rng *rand.Rand, goroutineID, messageID int) map[string]interface{} {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	services := []string{"auth", "api", "db", "cache", "queue"}
	
	return map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339Nano),
		"level":       levels[rng.Intn(len(levels))],
		"service":     services[rng.Intn(len(services))],
		"message":     fmt.Sprintf("Log message %d from goroutine %d", messageID, goroutineID),
		"goroutine":   goroutineID,
		"message_id":  messageID,
		"duration_ms": rng.Intn(1000),
		"metadata": map[string]interface{}{
			"host":       "server-" + strconv.Itoa(rng.Intn(10)),
			"version":    "1.0." + strconv.Itoa(rng.Intn(100)),
			"request_id": fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), rng.Intn(1000)),
		},
	}
}