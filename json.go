package serializer

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

// jsonSerializer implements Serializer using JSON encoding
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSON serializer
func NewJSONSerializer() Serializer {
	return &JSONSerializer{}
}

func (s *JSONSerializer) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (s *JSONSerializer) Deserialize(data []byte, v any) error {
	if data == nil {
		return errors.New("data is nil")
	}
	return json.Unmarshal(data, v)
}

func (s *JSONSerializer) SerializeTo(w io.Writer, v any) error {
	if w == nil {
		return errors.New("writer is nil")
	}
	return json.NewEncoder(w).Encode(v)
}

func (s *JSONSerializer) DeserializeFrom(r io.Reader, v any) error {
	if r == nil {
		return errors.New("reader is nil")
	}
	return json.NewDecoder(r).Decode(v)
}

func (s *JSONSerializer) ContentType() string {
	return "application/json"
}

func (s *JSONSerializer) GetType(data []byte) (Type, error) {
	if data == nil {
		return TypeNil, errors.New("data is nil")
	}

	// For JSON, we can use json.Unmarshal to determine the type
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return TypeNil, err
	}

	// Determine the type
	switch v.(type) {
	case string:
		return TypeString, nil
	case float64: // JSON numbers are always float64
		return TypeInt, nil
	case bool:
		return TypeBool, nil
	case []any:
		return TypeSlice, nil
	case map[string]any:
		return TypeMap, nil
	case nil:
		return TypeNil, nil
	default:
		// For structs, we need to check the type
		if reflect.TypeOf(v).Kind() == reflect.Struct {
			return TypeStruct, nil
		}
		return TypeNil, nil
	}
}
