package config

import (
	"encoding/base64"
	"fmt"
)

// encryptionKeyBytes is the size required by AES-256-GCM.
const encryptionKeyBytes = 32

// Security carries the cryptographic parameters.
type Security struct {
	// EncryptionKey must decode to exactly 32 bytes (AES-256-GCM).
	// Always a ${VAR} reference in the files, never a value.
	EncryptionKey string `yaml:"encryption_key"`
	Argon2        Argon2 `yaml:"argon2"`
}

// Argon2 carries the cost of password hashing.
type Argon2 struct {
	MemoryKiB  uint32 `yaml:"memory_kib"`
	Iterations uint32 `yaml:"iterations"`
	Threads    uint8  `yaml:"threads"`
}

// DecodedEncryptionKey decodes and validates the encryption key.
func (s Security) DecodedEncryptionKey() ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(s.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("security.encryption_key is not valid base64: %w", err)
	}
	if len(key) != encryptionKeyBytes {
		return nil, fmt.Errorf(
			"security.encryption_key must be %d bytes, got %d",
			encryptionKeyBytes, len(key),
		)
	}
	return key, nil
}
