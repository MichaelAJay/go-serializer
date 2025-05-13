package serializer

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
)

// JSONSerializer implements Serializer using JSON encoding
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSON serializer
func NewJSONSerializer() Serializer {
	return &JSONSerializer{}
}

func (s *JSONSerializer) Serialize(v any) ([]byte, error) {
	if v == nil {
		return nil, errors.New("cannot serialize nil value")
	}

	// Create a wrapper that includes type information
	wrapper := struct {
		Type  Type
		Value any
	}{
		Type:  s.getType(v),
		Value: v,
	}

	// Serialize the wrapper
	return json.Marshal(wrapper)
}

func (s *JSONSerializer) Deserialize(data []byte, v any) error {
	if data == nil {
		return errors.New("data is nil")
	}

	// First try to deserialize as a wrapper
	var wrapper struct {
		Type  Type
		Value any
	}

	if err := json.Unmarshal(data, &wrapper); err == nil {
		// If we successfully deserialized a wrapper, use the type information
		return s.deserializeWithType(wrapper.Value, wrapper.Type, v)
	}

	// If that fails, try direct deserialization
	return json.Unmarshal(data, v)
}

func (s *JSONSerializer) SerializeTo(w io.Writer, v any) error {
	if w == nil {
		return errors.New("writer is nil")
	}

	// Create a wrapper that includes type information
	wrapper := struct {
		Type  Type
		Value any
	}{
		Type:  s.getType(v),
		Value: v,
	}

	return json.NewEncoder(w).Encode(wrapper)
}

func (s *JSONSerializer) DeserializeFrom(r io.Reader, v any) error {
	if r == nil {
		return errors.New("reader is nil")
	}

	// First try to deserialize as a wrapper
	var wrapper struct {
		Type  Type
		Value any
	}

	if err := json.NewDecoder(r).Decode(&wrapper); err == nil {
		// If we successfully deserialized a wrapper, use the type information
		return s.deserializeWithType(wrapper.Value, wrapper.Type, v)
	}

	// If that fails, try direct deserialization
	return json.NewDecoder(r).Decode(v)
}

func (s *JSONSerializer) ContentType() string {
	return "application/json"
}

func (s *JSONSerializer) GetType(data []byte) (Type, error) {
	if data == nil {
		return TypeNil, errors.New("data is nil")
	}

	// Try to deserialize as a wrapper first
	var wrapper struct {
		Type  Type
		Value any
	}

	if err := json.Unmarshal(data, &wrapper); err == nil {
		return wrapper.Type, nil
	}

	// If that fails, try to determine type from the raw data
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return TypeNil, err
	}

	return s.getType(v), nil
}

// getType determines the type of a value
func (s *JSONSerializer) getType(v any) Type {
	if v == nil {
		return TypeNil
	}

	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return TypeInt
	case reflect.Float32, reflect.Float64:
		return TypeFloat
	case reflect.Bool:
		return TypeBool
	case reflect.Slice:
		return TypeSlice
	case reflect.Map:
		return TypeMap
	case reflect.Struct:
		return TypeStruct
	default:
		return TypeNil
	}
}

// deserializeWithType deserializes a value with type information
func (s *JSONSerializer) deserializeWithType(value any, valueType Type, target any) error {
	// Create a new value of the target type
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return errors.New("target must be a pointer")
	}
	targetType = targetType.Elem()

	// Convert the value to the target type
	switch valueType {
	case TypeString:
		if targetType.Kind() == reflect.String {
			reflect.ValueOf(target).Elem().SetString(value.(string))
			return nil
		}
	case TypeInt:
		if targetType.Kind() == reflect.Int || targetType.Kind() == reflect.Int64 {
			// JSON numbers are float64, so we need to convert
			reflect.ValueOf(target).Elem().SetInt(int64(value.(float64)))
			return nil
		}
	case TypeFloat:
		if targetType.Kind() == reflect.Float64 {
			reflect.ValueOf(target).Elem().SetFloat(value.(float64))
			return nil
		}
	case TypeBool:
		if targetType.Kind() == reflect.Bool {
			reflect.ValueOf(target).Elem().SetBool(value.(bool))
			return nil
		}
	case TypeSlice:
		if targetType.Kind() == reflect.Slice {
			// Handle slice conversion
			return s.convertSlice(value, target)
		}
	case TypeMap:
		if targetType.Kind() == reflect.Map {
			// Handle map conversion
			return s.convertMap(value, target)
		}
	case TypeStruct:
		if targetType.Kind() == reflect.Struct {
			// Handle struct conversion
			return s.convertStruct(value, target)
		}
	}

	// If we can't convert directly, try to marshal/unmarshal
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}

// convertSlice converts a slice value to the target type
func (s *JSONSerializer) convertSlice(value any, target any) error {
	// Implementation for slice conversion
	// This would handle converting between different slice types
	return json.Unmarshal(mustMarshal(value), target)
}

// convertMap converts a map value to the target type
func (s *JSONSerializer) convertMap(value any, target any) error {
	// Implementation for map conversion
	// This would handle converting between different map types
	return json.Unmarshal(mustMarshal(value), target)
}

// convertStruct converts a struct value to the target type
func (s *JSONSerializer) convertStruct(value any, target any) error {
	// Implementation for struct conversion
	// This would handle converting between different struct types
	return json.Unmarshal(mustMarshal(value), target)
}
