package serializer

import (
	"bytes"
	"errors"
	"io"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	// MAX_BUF_CAP is the maximum buffer capacity before we discard buffers from the pool
	// to prevent unbounded memory growth from large serializations
	MAX_BUF_CAP = 1 << 20 // 1MB
)

// pooledEncoder contains a reusable msgpack encoder and buffer
type pooledEncoder struct {
	enc *msgpack.Encoder
	buf *bytes.Buffer
}

// encoderPool is the global pool for reusing encoders and their buffers
var encoderPool = sync.Pool{
	New: func() any {
		buf := &bytes.Buffer{}
		return &pooledEncoder{
			enc: msgpack.NewEncoder(buf),
			buf: buf,
		}
	},
}

// getPooledEncoder retrieves a pooled encoder from the pool
func getPooledEncoder() *pooledEncoder {
	return encoderPool.Get().(*pooledEncoder)
}

// putPooledEncoder returns a pooled encoder to the pool
// If the buffer capacity exceeds MAX_BUF_CAP, the entire encoder is discarded to prevent memory bloat
func putPooledEncoder(pe *pooledEncoder) {
	if pe.buf.Cap() > MAX_BUF_CAP {
		// Discard the entire encoder - don't return it to the pool

		// @TODO - this needs observability
		return
	}
	encoderPool.Put(pe)
}

// pooledDecoder contains a reusable msgpack decoder and bytes reader
type pooledDecoder struct {
	dec    *msgpack.Decoder
	reader *bytes.Reader
}

// decoderPool is the global pool for reusing decoders and their readers
var decoderPool = sync.Pool{
	New: func() any {
		reader := bytes.NewReader(nil)
		return &pooledDecoder{
			dec:    msgpack.NewDecoder(reader),
			reader: reader,
		}
	},
}

// getPooledDecoder retrieves a pooled decoder from the pool and sets it up with the provided data
func getPooledDecoder(data []byte) *pooledDecoder {
	pd := decoderPool.Get().(*pooledDecoder)
	pd.reader.Reset(data)
	pd.dec.Reset(pd.reader)
	return pd
}

// putPooledDecoder returns a pooled decoder to the pool
func putPooledDecoder(pd *pooledDecoder) {
	// Reset the reader to nil to release reference to data
	pd.reader.Reset(nil)
	decoderPool.Put(pd)
}

// MsgPackSerializer implements Serializer using MessagePack encoding
type MsgPackSerializer struct{}

// NewMsgpackSerializer creates a new MessagePack serializer
func NewMsgpackSerializer() Serializer {
	return &MsgPackSerializer{}
}

// SerializeSafe uses pooled encoders to reduce allocations while returning an owned []byte slice.
// This provides the performance benefits of pooled encoders without requiring callers to manage buffer lifecycles.
func (s *MsgPackSerializer) SerializeSafe(v any) ([]byte, error) {
	if v == nil {
		return nil, errors.New("cannot serialize nil value")
	}

	// Acquire pooled encoder
	pe := getPooledEncoder()
	defer putPooledEncoder(pe)

	// Reset buffer and bind encoder to it
	pe.buf.Reset()
	pe.enc.Reset(pe.buf)

	// Encode the value
	if err := pe.enc.Encode(v); err != nil {
		return nil, err
	}

	// Copy to owned slice
	out := make([]byte, pe.buf.Len())
	copy(out, pe.buf.Bytes())

	return out, nil
}

func (s *MsgPackSerializer) Serialize(v any) ([]byte, error) {
	// Use SerializeSafe as the implementation to benefit from pooled encoders
	return s.SerializeSafe(v)
}

func (s *MsgPackSerializer) Deserialize(data []byte, v any) error {
	if data == nil {
		return errors.New("data is nil")
	}
	if v == nil {
		return errors.New("output parameter is nil")
	}

	// Use pooled decoder to reduce allocations
	pd := getPooledDecoder(data)
	defer putPooledDecoder(pd)

	return pd.dec.Decode(v)
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

// DeserializeString implements StringDeserializer interface
// Uses unsafe string-to-bytes conversion to avoid allocation
func (s *MsgPackSerializer) DeserializeString(data string, v any) error {
	if data == "" {
		return errors.New("data is empty")
	}
	return msgpack.Unmarshal(stringToReadOnlyBytes(data), v)
}

func (s *MsgPackSerializer) ContentType() string {
	return "application/x-msgpack"
}

// PooledBuf owns a pointer to an encoder's buffer. Caller must call Release()
// after the buffer is no longer needed to return the pooled encoder to the pool.
type PooledBuf struct {
	pe *pooledEncoder // holds the complete pooled encoder for release
}

// Bytes returns the encoded bytes from the pooled buffer.
// The returned slice is valid until Release() is called.
func (p *PooledBuf) Bytes() []byte {
	if p.pe == nil || p.pe.buf == nil {
		return nil
	}
	return p.pe.buf.Bytes()
}

// Len returns the length of the encoded data.
func (p *PooledBuf) Len() int {
	if p.pe == nil || p.pe.buf == nil {
		return 0
	}
	return p.pe.buf.Len()
}

// Release returns the underlying pooledEncoder back to the pool.
// After calling Release(), the PooledBuf should not be used anymore.
// The bytes returned by Bytes() become invalid after Release().
func (p *PooledBuf) Release() {
	if p.pe != nil {
		putPooledEncoder(p.pe)
		p.pe = nil // Prevent accidental reuse
	}
}

// SerializePooled encodes the value using a pooled encoder and returns a PooledBuf
// that provides zero-copy access to the encoded bytes. The caller MUST call Release()
// on the returned PooledBuf when done to return the encoder to the pool.
//
// This is the high-performance path that avoids copying the encoded bytes.
// Use this when you can guarantee that Release() will be called after all uses
// of the bytes are complete.
func (s *MsgPackSerializer) SerializePooled(v any) (*PooledBuf, error) {
	if v == nil {
		return nil, errors.New("cannot serialize nil value")
	}

	// Acquire pooled encoder
	pe := getPooledEncoder()

	// Reset buffer and bind encoder to it
	pe.buf.Reset()
	pe.enc.Reset(pe.buf)

	// Encode the value
	if err := pe.enc.Encode(v); err != nil {
		// On error, return encoder to pool immediately
		putPooledEncoder(pe)
		return nil, err
	}

	// Return PooledBuf with ownership of the encoder
	// Do NOT put the encoder back in the pool - ownership is transferred to caller
	return &PooledBuf{pe: pe}, nil
}

// DeserializeFromPooled decodes directly from a pooled buffer without copying the bytes.
// This provides zero-copy decoding when the data is already in a PooledBuf from SerializePooled.
// The PooledBuf is NOT released by this function - the caller remains responsible for calling Release().
func (s *MsgPackSerializer) DeserializeFromPooled(pb *PooledBuf, v any) error {
	if pb == nil {
		return errors.New("PooledBuf is nil")
	}
	if v == nil {
		return errors.New("output parameter is nil")
	}

	// Get bytes from the pooled buffer
	data := pb.Bytes()
	if data == nil {
		return errors.New("PooledBuf contains no data")
	}

	// Use pooled decoder to decode the data
	pd := getPooledDecoder(data)
	defer putPooledDecoder(pd)

	return pd.dec.Decode(v)
}

// CopyAndRelease is a convenience helper that copies the bytes from a PooledBuf
// to a fresh []byte slice, releases the pooled buffer, and returns the copy.
// This is useful when you want the performance benefits of pooled encoding
// but need an owned copy of the data.
func CopyAndRelease(pb *PooledBuf) []byte {
	if pb == nil {
		return nil
	}

	// Copy the bytes before releasing
	bytes := pb.Bytes()
	if bytes == nil {
		pb.Release() // Still release even if bytes is nil
		return nil
	}

	result := make([]byte, len(bytes))
	copy(result, bytes)

	// Release the pooled buffer
	pb.Release()

	return result
}
