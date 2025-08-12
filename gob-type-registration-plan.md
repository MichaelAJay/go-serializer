# Gob Serialization Type Registration Plan

## Problem Analysis

### Root Cause
The gob serialization issue occurs because Go's `encoding/gob` package requires type information to properly serialize/deserialize values. When you try to deserialize a concrete struct (like `Session`) into an `interface{}`, gob fails with:

```
gob: local interface type *interface {} can only be decoded from remote interface type; received concrete type Session
```

### What's Happening in the Cache

The go-cache memory provider has a "Progressive Type Recovery Algorithm" (lines 266-316 in memory.go) that attempts to handle deserialization failures:

1. **Primary attempt**: Deserialize to `interface{}`
2. **Fallback attempts**: Try primitive types (string, int, float64, bool)
3. **Problem**: It doesn't handle custom structs like `Session`

**What you're doing "wrong" (it's actually a design limitation):**
- The cache assumes it can always deserialize to `interface{}` 
- Gob requires concrete type knowledge for complex types
- The fallback only covers primitives, not structs
- Each deserialization failure triggers multiple retry attempts (performance overhead)

## Solution 1: Gob Type Registration (Recommended)

### Overview
Register known types with gob so it can handle `interface{}` deserialization properly.

### Implementation Plan

#### Step 1: Create Type Registry System
```go
// File: gob_registry.go
package serializer

import (
    "encoding/gob"
    "sync"
)

var (
    gobRegistry = &TypeRegistry{
        registered: make(map[string]bool),
    }
)

type TypeRegistry struct {
    mu         sync.RWMutex
    registered map[string]bool
}

func (r *TypeRegistry) Register(value interface{}) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    typeName := fmt.Sprintf("%T", value)
    if !r.registered[typeName] {
        gob.Register(value)
        r.registered[typeName] = true
    }
}

func (r *TypeRegistry) IsRegistered(value interface{}) bool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    typeName := fmt.Sprintf("%T", value)
    return r.registered[typeName]
}
```

#### Step 2: Enhance GobSerializer
```go
// Enhanced GobSerializer with auto-registration
type GobSerializer struct {
    autoRegister bool
}

func NewGobSerializer() Serializer {
    return &GobSerializer{autoRegister: true}
}

func NewGobSerializerWithOptions(autoRegister bool) Serializer {
    return &GobSerializer{autoRegister: autoRegister}
}

func (s *GobSerializer) Serialize(v any) ([]byte, error) {
    if v == nil {
        return nil, errors.New("cannot serialize nil value")
    }
    
    // Auto-register the type if enabled
    if s.autoRegister {
        gobRegistry.Register(v)
    }
    
    var buf bytes.Buffer
    encoder := gob.NewEncoder(&buf)
    err := encoder.Encode(v)
    return buf.Bytes(), err
}

// Deserialize remains the same - gob now knows about the types
func (s *GobSerializer) Deserialize(data []byte, v any) error {
    if data == nil {
        return errors.New("data is nil")
    }
    buf := bytes.NewBuffer(data)
    decoder := gob.NewDecoder(buf)
    return decoder.Decode(v)
}
```

#### Step 3: Pre-register Common Types
```go
// File: known_types.go
package serializer

func init() {
    // Register common types that will be used across modules
    RegisterCommonTypes()
}

func RegisterCommonTypes() {
    // Basic types
    gobRegistry.Register("")         // string
    gobRegistry.Register(int(0))     // int
    gobRegistry.Register(int64(0))   // int64
    gobRegistry.Register(float64(0)) // float64
    gobRegistry.Register(bool(false)) // bool
    
    // Common collection types
    gobRegistry.Register(map[string]string{})
    gobRegistry.Register(map[string]interface{}{})
    gobRegistry.Register([]string{})
    gobRegistry.Register([]interface{}{})
    
    // Time-related types
    gobRegistry.Register(time.Time{})
    gobRegistry.Register(time.Duration(0))
}

// Public API for external packages to register their types
func RegisterType(value interface{}) {
    gobRegistry.Register(value)
}

func RegisterTypes(values ...interface{}) {
    for _, v := range values {
        gobRegistry.Register(v)
    }
}
```

#### Step 4: Integration with go-auth
```go
// In go-auth initialization code (e.g., main.go or init())
import (
    "github.com/MichaelAJay/go-auth/models"
    "github.com/MichaelAJay/go-serializer"
)

func init() {
    // Register auth-specific types
    serializer.RegisterTypes(
        models.Session{},
        &models.Session{},
        models.SessionReplayProtection{},
        &models.SessionReplayProtection{},
    )
}
```

### Benefits of This Approach
- ✅ Fixes the deserialization issue completely
- ✅ No changes needed to go-cache
- ✅ Automatic type registration option
- ✅ Thread-safe implementation
- ✅ Backward compatible
- ✅ Performance improvement (eliminates fallback attempts)

### Potential Drawbacks
- ⚠️ Global state (gob registry is global)
- ⚠️ Must remember to register new types
- ⚠️ Auto-registration adds slight overhead

## Solution 2: Improve Cache Type Handling

### What's Wrong with Current Cache Approach

The "Progressive Type Recovery Algorithm" has these issues:

1. **Performance**: Multiple deserialization attempts on every failure
2. **Limited scope**: Only handles primitives, not structs
3. **Error masking**: Original errors get lost in fallback attempts
4. **Unpredictable**: Same data might deserialize to different types

### Proposed Cache Improvements

#### Option A: Type-Aware Serialization
Store type information alongside serialized data:

```go
type TypedValue struct {
    Type  string `json:"type"`
    Value []byte `json:"value"`
}

// During Set operation
func (c *memoryCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
    typeName := fmt.Sprintf("%T", value)
    
    // Serialize the actual value
    valueData, err := c.serializer.Serialize(value)
    if err != nil {
        return err
    }
    
    // Wrap with type information
    typedValue := TypedValue{
        Type:  typeName,
        Value: valueData,
    }
    
    // Serialize the wrapper
    data, err := c.serializer.Serialize(typedValue)
    // ... rest of Set logic
}

// During Get operation - use type info for proper deserialization
```

#### Option B: Configurable Deserialization Strategy
```go
type DeserializationStrategy int

const (
    StrategyInterface DeserializationStrategy = iota  // Current approach
    StrategyStrict                                   // Fail fast, no fallbacks
    StrategyTyped                                    // Use stored type info
)

func (c *memoryCache) SetDeserializationStrategy(strategy DeserializationStrategy) {
    c.deserializationStrategy = strategy
}
```

### Why Solution 1 (Gob Registration) is Better

The cache improvements would require:
- ❌ Breaking changes to cache storage format
- ❌ More complex serialization logic
- ❌ Migration path for existing data
- ❌ Performance overhead for type metadata

Gob registration fixes the root cause without any cache changes.

## Implementation Timeline

### Phase 1: Core Type Registry (1-2 hours)
1. Create `gob_registry.go` with thread-safe type registration
2. Create `known_types.go` with common type pre-registration
3. Add public API for external type registration

### Phase 2: Enhanced Serializer (1 hour)
1. Update `GobSerializer` with auto-registration option
2. Maintain backward compatibility
3. Add configuration options

### Phase 3: Integration (30 minutes)
1. Register `Session` and related types in go-auth
2. Update cache creation to use enhanced serializer
3. Test the integration

### Phase 4: Documentation & Testing (1 hour)
1. Document type registration patterns
2. Add unit tests for type registry
3. Add integration tests with various struct types

## Testing Strategy

### Unit Tests
```go
func TestGobTypeRegistration(t *testing.T) {
    // Test auto-registration
    serializer := NewGobSerializer()
    
    session := &models.Session{ID: "test"}
    
    // Serialize
    data, err := serializer.Serialize(session)
    require.NoError(t, err)
    
    // Deserialize to interface{}
    var result interface{}
    err = serializer.Deserialize(data, &result)
    require.NoError(t, err)
    
    // Should be able to type assert back to Session
    recoveredSession, ok := result.(*models.Session)
    require.True(t, ok)
    assert.Equal(t, "test", recoveredSession.ID)
}
```

### Integration Tests
```go
func TestCacheWithComplexTypes(t *testing.T) {
    // Test that cache can handle Session structs without fallback
    cache := createCacheWithGobSerializer(t)
    
    session := createTestSession("test-id", "user-123")
    
    // Store session
    err := cache.Set(ctx, "session:test", session, time.Hour)
    require.NoError(t, err)
    
    // Retrieve session
    value, found, err := cache.Get(ctx, "session:test")
    require.NoError(t, err)
    require.True(t, found)
    
    // Should get back the exact same type
    retrievedSession, ok := value.(*models.Session)
    require.True(t, ok)
    assert.Equal(t, session.ID, retrievedSession.ID)
}
```

## Conclusion

**Recommended approach**: Implement Solution 1 (Gob Type Registration) because:

1. **Root cause fix**: Addresses the fundamental gob serialization limitation
2. **No breaking changes**: Existing code continues to work
3. **Performance improvement**: Eliminates expensive fallback attempts
4. **Simple implementation**: Minimal code changes required
5. **Future-proof**: Works for any struct type you need to cache

The current cache implementation isn't "wrong" per se - it's a reasonable attempt to handle gob's limitations. But type registration is a cleaner, more efficient solution that handles the problem at its source.