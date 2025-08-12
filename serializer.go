package serializer

import (
	"fmt"
	"io"
	"reflect"
)

// Serializer interface defines the contract for serialization implementations
type Serializer interface {
	// Serialize converts a value to bytes
	Serialize(v any) ([]byte, error)

	// Deserialize converts bytes back to a value
	// v must be a pointer to the type you want to deserialize into
	Deserialize(data []byte, v any) error

	// SerializeTo writes a value to a writer
	SerializeTo(w io.Writer, v any) error

	// DeserializeFrom reads a value from a reader
	// v must be a pointer to the type you want to deserialize into
	DeserializeFrom(r io.Reader, v any) error

	// ContentType returns the MIME type for this serialization format
	ContentType() string
}

// TypeInfo holds runtime type information for typed serialization
type TypeInfo struct {
	Type     reflect.Type
	TypeName string
}

// TypedSerializer extends Serializer with type-aware operations
// This allows the serializer to know the exact target type for deserialization
type TypedSerializer interface {
	Serializer
	
	// SerializeWithTypeInfo optimizes serialization based on type information
	SerializeWithTypeInfo(v any, typeInfo TypeInfo) ([]byte, error)
	
	// DeserializeWithTypeInfo uses type information to deserialize to the exact type
	// This is crucial for gob serialization which needs concrete type information
	DeserializeWithTypeInfo(data []byte, typeInfo TypeInfo) (any, error)
}

// Format enum for selecting serializers
type Format string

const (
	JSON    Format = "json"
	Binary  Format = "binary"
	Msgpack Format = "msgpack"
)

// Registry for managing serializers
type Registry struct {
	serializers map[Format]Serializer
}

// NewRegistry creates a new serializer registry
func NewRegistry() *Registry {
	return &Registry{
		serializers: make(map[Format]Serializer),
	}
}

// Register adds a serializer to the registry
func (r *Registry) Register(format Format, serializer Serializer) {
	r.serializers[format] = serializer
}

// Get retrieves a serializer from the registry
func (r *Registry) Get(format Format) (Serializer, bool) {
	serializer, ok := r.serializers[format]
	return serializer, ok
}

// New creates a new serializer instance
func (r *Registry) New(format Format) (Serializer, error) {
	serializer, ok := r.serializers[format]
	if !ok {
		return nil, fmt.Errorf("serializer for format %s not found", format)
	}
	return serializer, nil
}

// RegisterDefaultSerializers registers all available serializers
func RegisterDefaultSerializers() {
	DefaultRegistry.Register(JSON, NewJSONSerializer())
	DefaultRegistry.Register(Binary, NewGobSerializer())
	DefaultRegistry.Register(Msgpack, NewMsgpackSerializer())
}

// Initialize default serializers
func init() {
	RegisterDefaultSerializers()
}
