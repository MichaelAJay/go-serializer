package serializer

import (
	"bytes"
	"encoding/gob"
	"io"
)

type GobSerializer struct{}

func (s *GobSerializer) Serialize(v any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(v)
	return buf.Bytes(), err
}

func (s *GobSerializer) Deserialize(data []byte, v any) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	return decoder.Decode(v)
}

func (s *GobSerializer) SerializeTo(w io.Writer, v any) error {
	encoder := gob.NewEncoder(w)
	return encoder.Encode(v)
}

func (s *GobSerializer) DeserializeFrom(r io.Reader, v any) error {
	decoder := gob.NewDecoder(r)
	return decoder.Decode(v)
}

func (s *GobSerializer) ContentType() string {
	return "application/octet-stream"
}
