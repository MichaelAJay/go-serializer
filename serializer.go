package serializer

import (
	"fmt"
	"io"
)

// Type represents the type of a serialized value
type Type string

const (
	TypeString Type = "string"
	TypeInt    Type = "int"
	TypeFloat  Type = "float"
	TypeBool   Type = "bool"
	TypeSlice  Type = "slice"
	TypeMap    Type = "map"
	TypeStruct Type = "struct"
	TypeNil    Type = "nil"
)

// SerializedValue represents a value that has been serialized
type SerializedValue struct {
	Type  Type
	Value any
}

// Serializer interface defines the contract for serialization implementations
type Serializer interface {
	// Serialize converts a value to bytes
	// It should preserve type information and ensure uniform representation
	Serialize(v any) ([]byte, error)

	// Deserialize converts bytes back to a value
	// It should restore the original type structure and ensure uniform representation
	// The output should be consistent across all serializer implementations
	Deserialize(data []byte, v any) error

	// SerializeTo writes a value to a writer
	// It should preserve type information and ensure uniform representation
	SerializeTo(w io.Writer, v any) error

	// DeserializeFrom reads a value from a reader
	// It should restore the original type structure and ensure uniform representation
	DeserializeFrom(r io.Reader, v any) error

	// ContentType returns the MIME type for this serialization format
	ContentType() string

	// GetType returns the type of a serialized value
	GetType(data []byte) (Type, error)
}

// Format enum for selecting serializers
type Format string

const (
	JSON    Format = "json"
	Binary  Format = "binary"
	Proto   Format = "protobuf"
	Msgpack Format = "msgpack"
	CBOR    Format = "cbor"
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
