package serializer

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
)

// registeredTypes tracks types that have been registered with gob
// We track by the base type (element type for pointers) to avoid conflicts
var (
	registeredTypes = make(map[reflect.Type]bool)
	registrationMu  sync.RWMutex
)

// GobSerializer implements Serializer using Gob encoding
type GobSerializer struct{}

// NewGobSerializer creates a new Gob serializer
func NewGobSerializer() Serializer {
	return &GobSerializer{}
}

func (s *GobSerializer) Serialize(v any) ([]byte, error) {
	if v == nil {
		return nil, errors.New("cannot serialize nil value")
	}
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(v)
	return buf.Bytes(), err
}

func (s *GobSerializer) Deserialize(data []byte, v any) error {
	if data == nil {
		return errors.New("data is nil")
	}
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(v)
}

func (s *GobSerializer) SerializeTo(w io.Writer, v any) error {
	if w == nil {
		return errors.New("writer is nil")
	}
	encoder := gob.NewEncoder(w)
	return encoder.Encode(v)
}

func (s *GobSerializer) DeserializeFrom(r io.Reader, v any) error {
	if r == nil {
		return errors.New("reader is nil")
	}
	decoder := gob.NewDecoder(r)
	return decoder.Decode(v)
}

func (s *GobSerializer) ContentType() string {
	return "application/x-gob"
}

// SerializeWithTypeInfo implements TypedSerializer interface
// For gob serialization, this ensures type registration and provides better error context
func (s *GobSerializer) SerializeWithTypeInfo(v any, typeInfo TypeInfo) ([]byte, error) {
	if v == nil {
		return nil, errors.New("cannot serialize nil value")
	}
	
	// Automatically register the type with gob
	if typeInfo.Type != nil {
		registerTypeIfNeeded(typeInfo.Type)
	}
	
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("gob serialization failed for type %s: %w", typeInfo.TypeName, err)
	}
	return buf.Bytes(), nil
}

// registerTypeIfNeeded ensures the type is registered with gob
// We register based on the base type to avoid pointer/value conflicts
func registerTypeIfNeeded(t reflect.Type) {
	// Get the base type (element type for pointers)
	baseType := t
	if t.Kind() == reflect.Ptr {
		baseType = t.Elem()
	}

	registrationMu.RLock()
	if registeredTypes[baseType] {
		registrationMu.RUnlock()
		return
	}
	registrationMu.RUnlock()

	registrationMu.Lock()
	defer registrationMu.Unlock()
	
	// Double-check after acquiring write lock
	if registeredTypes[baseType] {
		return
	}
	
	// Register the base type (as a value) - gob can handle both pointer and value forms
	// when the value type is registered
	zeroValue := reflect.New(baseType).Elem().Interface()
	gob.Register(zeroValue)
	
	registeredTypes[baseType] = true
}

// DeserializeWithTypeInfo implements TypedSerializer interface
// This is the key method that solves gob deserialization issues
func (s *GobSerializer) DeserializeWithTypeInfo(data []byte, typeInfo TypeInfo) (any, error) {
	if data == nil {
		return nil, errors.New("data is nil")
	}
	
	if typeInfo.Type == nil {
		return nil, errors.New("typeInfo.Type is nil")
	}
	
	// Automatically register the type with gob
	registerTypeIfNeeded(typeInfo.Type)
	
	// Create a new instance of the target type
	// This gives gob the concrete type it needs for deserialization
	targetValue := reflect.New(typeInfo.Type)
	
	// Handle pointer types
	var deserializeTarget any
	if typeInfo.Type.Kind() == reflect.Ptr {
		// For pointer types, we need to create the underlying type
		elemType := typeInfo.Type.Elem()
		elemValue := reflect.New(elemType)
		targetValue.Elem().Set(elemValue)
		deserializeTarget = targetValue.Interface()
	} else {
		// For non-pointer types, use the pointer to the new instance
		deserializeTarget = targetValue.Interface()
	}
	
	// Deserialize using the concrete type
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(deserializeTarget)
	if err != nil {
		return nil, fmt.Errorf("gob deserialization failed for type %s: %w (hint: check for pointer/value type mismatches)", typeInfo.TypeName, err)
	}
	
	// Return the correct value based on the original type
	if typeInfo.Type.Kind() == reflect.Ptr {
		// For pointer types, return the pointer
		return targetValue.Elem().Interface(), nil
	} else {
		// For non-pointer types, return the dereferenced value
		return targetValue.Elem().Interface(), nil
	}
}
