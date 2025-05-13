package serializer

import (
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

type MsgPackSerializer struct{}

func (s *MsgPackSerializer) Serialize(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (s *MsgPackSerializer) Deserialize(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}

func (s *MsgPackSerializer) SerializeTo(w io.Writer, v any) error {
	return msgpack.NewEncoder(w).Encode(v)
}

func (s *MsgPackSerializer) DeserializeFrom(r io.Reader, v any) error {
	return msgpack.NewDecoder(r).Decode(v)
}

func (s *MsgPackSerializer) ContentType() string {
	return "application/msgpack"
}
