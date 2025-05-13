package serializer

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
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
