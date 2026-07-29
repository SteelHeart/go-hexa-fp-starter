package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// aesKeyLen enforces AES-256, and nothing else.
//
// `aes.NewCipher` accepts 16, 24 or 32 bytes: a 16-byte key silently produces
// AES-128. The fallback is invisible — everything encrypts and decrypts
// normally — whereas the guarantee announced by this package, and by the
// documentation that relies on it, is AES-256.
//
// This is a fail-open: you believe you have 256 bits, you have 128. Deny by
// default requires refusing rather than degrading.
const aesKeyLen = 32

// ErrInvalidKey reports an encryption key of the wrong length.
var ErrInvalidKey = errors.New("invalid encryption key")

// Cipher encrypts and decrypts data at rest with AES-256-GCM.
type Cipher struct{ aead cipher.AEAD }

// NewCipher builds a cipher from a 32-byte key.
//
// Any other length is REFUSED, including those AES accepts: see aesKeyLen.
func NewCipher(key []byte) (Cipher, error) {
	if len(key) != aesKeyLen {
		return Cipher{}, fmt.Errorf(
			"%w: %d bytes supplied, %d expected (AES-256)", ErrInvalidKey, len(key), aesKeyLen)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Cipher{}, fmt.Errorf("invalid AES key: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return Cipher{}, fmt.Errorf("GCM initialisation: %w", err)
	}
	return Cipher{aead: aead}, nil
}

// Encrypt encrypts and authenticates. The nonce is random and prefixed to the
// message: reusing a nonce with GCM reveals the content, so it is never derived.
func (c Cipher) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("nonce generation: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts and checks authenticity.
func (c Cipher) Decrypt(encoded string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("unreadable ciphertext: %w", err)
	}
	size := c.aead.NonceSize()
	if len(raw) < size {
		return nil, errors.New("ciphertext too short")
	}
	plaintext, err := c.aead.Open(nil, raw[:size], raw[size:], nil)
	if err != nil {
		// Message deliberately mute: distinguishing "wrong key" from
		// "tampered message" would help an attacker.
		return nil, errors.New("decryption refused")
	}
	return plaintext, nil
}
