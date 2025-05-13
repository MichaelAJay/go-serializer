package serializer

import (
	"errors"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

type MsgPackSerializer struct{}

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
