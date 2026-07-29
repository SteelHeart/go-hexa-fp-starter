package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestEncryptThenDecryptRoundTrips: the nominal path of encryption at rest.
//
// Encrypted data that can no longer be decrypted is LOST data, with no error
// message at the moment it is lost. This is the only test that prevents it.
func TestEncryptThenDecryptRoundTrips(t *testing.T) {
	t.Parallel()

	cipher, err := security.NewCipher(aesKey())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	for _, plaintext := range []string{
		"", "a", "some personal data", "accents éàü and symbols ✓",
	} {
		ciphertext, err := cipher.Encrypt([]byte(plaintext))
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plaintext, err)
		}
		decrypted, err := cipher.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", plaintext, err)
		}
		if string(decrypted) != plaintext {
			t.Errorf("round trip = %q, want %q", decrypted, plaintext)
		}
	}
}
