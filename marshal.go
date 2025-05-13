package serializer

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"
)

// mustMarshal is a helper function that panics if marshaling fails.
// It's used internally by serializers for type conversion.
func mustMarshal(v any) []byte {
	// Try JSON first as it's more commonly used
	bytes, err := json.Marshal(v)
	if err == nil {
		return bytes
	}

	// Fall back to MessagePack if JSON fails
	bytes, err = msgpack.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bytes
}
