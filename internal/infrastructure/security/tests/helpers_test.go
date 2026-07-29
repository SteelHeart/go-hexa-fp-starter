// Package tests contains the BLACK BOX tests of the security primitives: they
// only use the public API, exactly like an adapter.
//
// Repository convention (rules/tests.md): `{package}/tests/` for the black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is checked, without having to open it.
//
// # Why this package deserves more tests than the others
//
// A hashing or encryption defect NEVER shows up in use: everything keeps
// working. A malformed digest still verifies, a message encrypted with a reused
// nonce still decrypts. The defect only appears the day someone exploits it —
// that is, too late, and without warning.
package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// testParams is a DELIBERATELY low cost.
//
// Production parameters are designed to be slow: that is their job. Using them
// here would make the suite unusable, and a slow suite ends up never being run.
// The behaviour under test — format, bounds, comparison — does not depend on the
// cost.
func testParams() security.Argon2Params {
	return security.Argon2Params{MemoryKiB: 1 << 10, Iterations: 1, Threads: 1}
}

// newHasher builds a test hasher.
func newHasher() security.Hasher { return security.NewHasher(testParams()) }

// hash produces a digest, or fails the test.
func hash(t *testing.T, plain string) string {
	t.Helper()
	encoded, err := newHasher().Hash(plain)
	if err != nil {
		t.Fatalf("Hash(%q): %v", plain, err)
	}
	return encoded
}

// aesKey returns a deterministic 32-byte key.
//
// Built at run time, never hard-coded: rules/securite.md forbids any versioned
// secret, and a 32-byte string in a file is indistinguishable from a real leak,
// for gitleaks as much as for a reader.
func aesKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

// clone copies a slice, so as to tamper without touching the original.
func clone(src []byte) []byte { return append([]byte(nil), src...) }
