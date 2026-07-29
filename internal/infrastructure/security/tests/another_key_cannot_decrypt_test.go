package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestAnotherKeyCannotDecrypt: another key does not decrypt.
//
// Obvious on paper, decisive in operation: it is what makes a key rotation
// visible. If a wrong key still decrypted, a failed rotation would only be
// noticed once the data had become unreadable for everyone — long after the
// faulty deployment.
func TestAnotherKeyCannotDecrypt(t *testing.T) {
	t.Parallel()

	original, err := security.NewCipher(aesKey())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	ciphertext, err := original.Encrypt([]byte("some personal data"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	otherKey := aesKey()
	otherKey[0] ^= 0xFF
	other, err := security.NewCipher(otherKey)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	if _, err := other.Decrypt(ciphertext); err == nil {
		t.Error("another key must NOT be able to decrypt")
	}
}
