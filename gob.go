package serializer

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"reflect"
)

type GobSerializer struct{}

// NewGobSerializer creates a new Gob serializer
func NewGobSerializer() Serializer {
	return &GobSerializer{}
}

func (s *GobSerializer) Serialize(v any) ([]byte, error) {
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
	return "application/octet-stream"
}

func (s *GobSerializer) GetType(data []byte) (Type, error) {
	if data == nil {
		return TypeNil, errors.New("data is nil")
	}

	// For gob, we need to decode into an interface{} first
	var v any
	if err := s.Deserialize(data, &v); err != nil {
		return TypeNil, err
	}

	// Determine the type
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return TypeString, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return TypeInt, nil
	case reflect.Float32, reflect.Float64:
		return TypeFloat, nil
	case reflect.Bool:
		return TypeBool, nil
	case reflect.Slice:
		return TypeSlice, nil
	case reflect.Map:
		return TypeMap, nil
	case reflect.Struct:
		return TypeStruct, nil
	default:
		return TypeNil, nil
	}
}
