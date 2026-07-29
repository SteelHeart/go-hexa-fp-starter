package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestEachEncryptionUsesAFreshNonce: two encryptions of the SAME plaintext give
// two different messages.
//
// With GCM, reusing a nonce across two messages does not merely weaken the
// encryption: it reveals the XOR of the two plaintexts, and compromises the
// authentication key. It is the gravest fault possible on this mode, and it is
// undetectable in use — both messages decrypt perfectly.
//
// That is why the nonce is RANDOM and never derived from an identifier, however
// tempting it may be to make it reproducible.
func TestEachEncryptionUsesAFreshNonce(t *testing.T) {
	t.Parallel()

	cipher, err := security.NewCipher(aesKey())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	const plaintext = "some personal data"
	seen := map[string]struct{}{}
	for range 5 {
		ciphertext, err := cipher.Encrypt([]byte(plaintext))
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		if _, already := seen[ciphertext]; already {
			t.Fatal("two encryptions of the same plaintext produced the same message: nonce reused")
		}
		seen[ciphertext] = struct{}{}
	}
}
