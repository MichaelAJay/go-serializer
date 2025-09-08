//go:build go1.20

package serializer

import (
	"unsafe"
)

// stringToReadOnlyBytes converts a string to a read-only []byte slice using unsafe.
// This avoids the allocation that would occur with []byte(s).
//
// SAFETY REQUIREMENTS:
// - The returned []byte MUST NOT be modified
// - The returned slice is valid only as long as the original string exists
// - Modifying the returned slice will cause undefined behavior
// - This function is safe for read-only operations like json.Unmarshal, msgpack.Unmarshal
//
// The implementation uses Go 1.20+ unsafe.Slice() which is safer than previous
// unsafe string conversion techniques as it properly handles the slice header.
func stringToReadOnlyBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	// unsafe.StringData returns a pointer to the underlying string data
	// unsafe.Slice creates a slice from the pointer with the specified length
	return unsafe.Slice(unsafe.StringData(s), len(s))
}