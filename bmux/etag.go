package bmux

import (
	"strconv"

	"github.com/akyoto/hash"
)

// ETag produces a hash for the given slice of bytes.
// It is the same hash that Aero uses for its ETag header.
func ETag(b []byte) string {
	return strconv.FormatUint(hash.Bytes(b), 16)
}
