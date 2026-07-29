package tests

import (
	"encoding/base64"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestTamperedCiphertextIsRefused: tampering with a single bit makes decryption
// fail.
//
// This is the difference between encrypting and AUTHENTICATING. A mode without
// authentication would decrypt a modified message without flinching, and would
// return a silently wrong plaintext — an amount, an access right, an identifier.
// GCM refuses, and this test proves it rather than assuming it.
func TestTamperedCiphertextIsRefused(t *testing.T) {
	t.Parallel()

	cipher, err := security.NewCipher(aesKey())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	ciphertext, err := cipher.Encrypt([]byte("some personal data"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// The ciphertext travels base64-encoded: we tamper with the BYTES then
	// re-encode, so as to target the message itself and not its textual form.
	raw, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("the ciphertext must be decodable: %v", err)
	}

	cases := map[string]func([]byte) []byte{
		"a bit of the message": func(c []byte) []byte { d := clone(c); d[len(d)-1] ^= 1; return d },
		"a bit of the nonce":   func(c []byte) []byte { d := clone(c); d[0] ^= 1; return d },
		"truncated message":    func(c []byte) []byte { return c[:len(c)-1] },
		"empty message":        func([]byte) []byte { return nil },
		"too short":            func([]byte) []byte { return []byte{1, 2, 3} },
	}

	for name, tamper := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			forged := base64.RawURLEncoding.EncodeToString(tamper(raw))
			if _, err := cipher.Decrypt(forged); err == nil {
				t.Error("a tampered message must be REFUSED, never decrypted")
			}
		})
	}

	// A ciphertext that is not even base64 must be refused as well.
	if _, err := cipher.Decrypt("!!! not base64 !!!"); err == nil {
		t.Error("an unreadable ciphertext must be refused")
	}
}
