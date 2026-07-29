package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameters of the Argon2id encoding format.
const (
	argon2SaltLen = 16
	argon2KeyLen  = 32
	argon2Version = 19
	argon2Parts   = 6
)

// Bounds of the decoded digest.
//
// A digest is supposed to come from our own database, but nothing guarantees it:
// a data import, a badly migrated column, an external store are enough to slip
// something else in. Without a bound, the announced key length directly drives
// an allocation in argon2.IDKey — a forged "digest" announcing four gigabytes
// would bring the process down on the first verification.
//
// The bounds are generous so as not to invalidate a legitimate digest produced
// with another key length, while still closing the absurd case.
const (
	argon2MinKeyLen = 16
	argon2MaxKeyLen = 64
)

// ErrInvalidHash reports an unreadable digest: unknown format, unexpected
// version, or corrupted encoding.
var ErrInvalidHash = errors.New("invalid password digest")

// Argon2Params carries the hashing cost.
//
// Raising Memory is what really protects against massively parallel computation:
// raising Iterations alone costs an attacker on GPU as much as it costs the
// server.
type Argon2Params struct {
	MemoryKiB  uint32
	Iterations uint32
	Threads    uint8
}

// Hasher hashes and verifies passwords with Argon2id.
type Hasher struct{ params Argon2Params }

// NewHasher builds a hasher.
func NewHasher(params Argon2Params) Hasher { return Hasher{params: params} }

// Hash produces a self-describing digest: the format embeds the version and the
// parameters, which allows the cost to be raised without invalidating existing
// digests.
//
//	$argon2id$v=19$m=65536,t=3,p=4$<salt b64>$<key b64>
func (h Hasher) Hash(plain string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt generation: %w", err)
	}
	key := argon2.IDKey(
		[]byte(plain), salt,
		h.params.Iterations, h.params.MemoryKiB, h.params.Threads, argon2KeyLen,
	)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2Version, h.params.MemoryKiB, h.params.Iterations, h.params.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify compares a plaintext password against a digest, in constant time.
//
// A format error and an incorrect password are two different things: the first
// is a data defect, the second a nominal case.
func (h Hasher) Verify(plain, encoded string) (bool, error) {
	decoded, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	// The conversion is safe: decodeHash has already refused any out-of-bounds length.
	keyLen := uint32(len(decoded.key)) //nolint:gosec // bounded by decodeHash
	candidate := argon2.IDKey(
		[]byte(plain), decoded.salt,
		decoded.params.Iterations, decoded.params.MemoryKiB, decoded.params.Threads, keyLen,
	)
	return subtle.ConstantTimeCompare(decoded.key, candidate) == 1, nil
}

// NeedsRehash reports whether a digest was produced with a cost lower than the
// current one. To be called after a successful verification, so as to raise the
// cost transparently.
func (h Hasher) NeedsRehash(encoded string) bool {
	decoded, err := decodeHash(encoded)
	if err != nil {
		// An unreadable digest deserves to be redone, not kept.
		return true
	}
	return decoded.params.MemoryKiB < h.params.MemoryKiB ||
		decoded.params.Iterations < h.params.Iterations
}

// decodedHash carries the pieces of a digest: the parameters, the salt and the
// key.
//
// Grouped into a type rather than separate return values: they only mean
// something together — verifying a key with the salt of another digest means
// nothing — and two returns of the same shape (`[]byte`, `[]byte`) get swapped
// without the compiler flinching.
//
// ⚠️ This comment announced "three pieces" and then listed two. The count was
// wrong, and it was wrong in the documentation of the very type born from the
// "more than two returns = a missing type" rule.
type decodedHash struct {
	params Argon2Params
	salt   []byte
	key    []byte
}

func decodeHash(encoded string) (decodedHash, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != argon2Parts || parts[1] != "argon2id" {
		return decodedHash{}, fmt.Errorf("%w: prefix", ErrInvalidHash)
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2Version {
		return decodedHash{}, fmt.Errorf("%w: version", ErrInvalidHash)
	}
	var params Argon2Params
	if _, err := fmt.Sscanf(
		parts[3], "m=%d,t=%d,p=%d",
		&params.MemoryKiB, &params.Iterations, &params.Threads,
	); err != nil {
		return decodedHash{}, fmt.Errorf("%w: parameters", ErrInvalidHash)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return decodedHash{}, fmt.Errorf("%w: salt", ErrInvalidHash)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return decodedHash{}, fmt.Errorf("%w: key", ErrInvalidHash)
	}
	if len(key) < argon2MinKeyLen || len(key) > argon2MaxKeyLen {
		return decodedHash{}, fmt.Errorf("%w: key length (%d bytes)", ErrInvalidHash, len(key))
	}
	return decodedHash{params: params, salt: salt, key: key}, nil
}
