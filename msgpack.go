package serializer

import (
	"errors"
	"io"
	"reflect"

	"github.com/vmihailenco/msgpack/v5"
)

type MsgPackSerializer struct{}

// NewMsgpackSerializer creates a new MessagePack serializer
func NewMsgpackSerializer() Serializer {
	return &MsgPackSerializer{}
}

func (s *MsgPackSerializer) Serialize(v any) ([]byte, error) {
	if v == nil {
		return nil, errors.New("value is nil")
	}
	return msgpack.Marshal(v)
}

func (s *MsgPackSerializer) Deserialize(data []byte, v any) error {
	if data == nil {
		return errors.New("data is nil")
	}
	return msgpack.Unmarshal(data, v)
}

func (s *MsgPackSerializer) SerializeTo(w io.Writer, v any) error {
	if w == nil {
		return errors.New("writer is nil")
	}
	return msgpack.NewEncoder(w).Encode(v)
}

func (s *MsgPackSerializer) DeserializeFrom(r io.Reader, v any) error {
	if r == nil {
		return errors.New("reader is nil")
	}
	return msgpack.NewDecoder(r).Decode(v)
}

func (s *MsgPackSerializer) ContentType() string {
	return "application/msgpack"
}

func (s *MsgPackSerializer) GetType(data []byte) (Type, error) {
	if data == nil {
		return TypeNil, errors.New("data is nil")
	}

	// First try to unmarshal to a map to check the type
	var m map[string]any
	if err := msgpack.Unmarshal(data, &m); err == nil {
		return TypeMap, nil
	}

	// Try other basic types
	var v any
	if err := msgpack.Unmarshal(data, &v); err != nil {
		return TypeNil, err
	}

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
	case reflect.Struct:
		return TypeStruct, nil
	default:
		return TypeNil, nil
	}
}
