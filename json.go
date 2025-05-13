package serializer

import (
	"encoding/json"
	"io"
)

type JSONSerializer struct{}

func (s *JSONSerializer) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (s *JSONSerializer) Deserialize(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (s *JSONSerializer) SerializeTo(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func (s *JSONSerializer) DeserializeFrom(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

func (s *JSONSerializer) ContentType() string {
	return "application/json"
}
