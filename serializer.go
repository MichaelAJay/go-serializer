package serializer

import (
	"fmt"
	"io"
)

type Serializer interface {
	Serialize(v any) ([]byte, error)
	// v should be a pointer to the value to be deserialized
	Deserialize(data []byte, v any) error

	// For streaming operations
	SerializeTo(w io.Writer, v any) error
	DeserializeFrom(r io.Reader, v any) error

	// content type for HTTP headers
	ContentType() string
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

// Registery for managing serializers
type Registry struct {
	serializers map[Format]Serializer
}

func NewRegistry() *Registry {
	return &Registry{
		serializers: make(map[Format]Serializer),
	}
}

func (r *Registry) Register(format Format, serializer Serializer) {
	r.serializers[format] = serializer
}

func (r *Registry) Get(format Format) (Serializer, bool) {
	serializer, ok := r.serializers[format]
	return serializer, ok
}

// Factory function for creating serializers
func (r *Registry) New(format Format) (Serializer, error) {
	serializer, ok := r.serializers[format]
	if !ok {
		return nil, fmt.Errorf("serializer for format %s not found", format)
	}
	return serializer, nil
}
