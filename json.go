package serializer

import (
	"bytes"
	"errors"
	"io"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigFastest

type pooledBufferPool struct {
	pool          sync.Pool
	maxBufferSize int
}

func newPooledBufferPool(maxSize int) *pooledBufferPool {
	return &pooledBufferPool{
		pool: sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
		maxBufferSize: maxSize,
	}
}

func (p *pooledBufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *pooledBufferPool) Put(buf *bytes.Buffer) {
	if p.maxBufferSize > 0 && buf.Cap() > p.maxBufferSize {
		return
	}

	buf.Reset() // ensure no data lingers in memory
	p.pool.Put(buf)
}

// JSONSerializer implements Serializer using JSON encoding
type JSONSerializer struct {
	bufferPool *pooledBufferPool
}

// NewJSONSerializer creates a new JSON serializer
// If maxBufferSize <= 0, buffers are never capped.
func NewJSONSerializer(maxBufferSize int) Serializer {
	return &JSONSerializer{
		bufferPool: newPooledBufferPool(maxBufferSize),
	}
}

func (s *JSONSerializer) Serialize(v any) ([]byte, error) {
	if v == nil {
		return nil, errors.New("cannot serialize nil value")
	}

	buf := s.bufferPool.Get()
	defer s.bufferPool.Put(buf)

	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	data := make([]byte, buf.Len())
	copy(data, buf.Bytes())

	return data, nil
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
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func (s *JSONSerializer) DeserializeFrom(r io.Reader, v any) error {
	if r == nil {
		return errors.New("reader is nil")
	}
	return json.NewDecoder(r).Decode(v)
}

// DeserializeString implements StringDeserializer interface
// Uses unsafe string-to-bytes conversion to avoid allocation
func (s *JSONSerializer) DeserializeString(data string, v any) error {
	if data == "" {
		return errors.New("data is empty")
	}
	return json.Unmarshal(stringToReadOnlyBytes(data), v)
}

func (s *JSONSerializer) ContentType() string {
	return "application/json"
}
