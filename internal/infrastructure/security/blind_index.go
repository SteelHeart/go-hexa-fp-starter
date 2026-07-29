package security

import (
	"crypto/sha256"
	"encoding/base64"
)

// BlindIndex produces a deterministic index that allows an encrypted value to
// be searched for without decrypting it.
//
// ⚠️ Deterministic, therefore it leaks equality: two identical values have the
// same index. Acceptable for an exact search on a high-entropy field, never for
// a low-cardinality field.
func BlindIndex(key, value []byte) string {
	mac := sha256.Sum256(append(append([]byte(nil), key...), value...))
	return base64.RawURLEncoding.EncodeToString(mac[:])
}
