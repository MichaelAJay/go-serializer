package serializer

import (
	"encoding/json"
	"errors"
	"io"
)

type JSONSerializer struct{}

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
