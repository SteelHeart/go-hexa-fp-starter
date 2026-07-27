package config

import (
	"encoding/base64"
	"fmt"
)

// encryptionKeyBytes est la taille exigée par AES-256-GCM.
const encryptionKeyBytes = 32

// Security porte les paramètres cryptographiques.
type Security struct {
	// EncryptionKey doit décoder sur exactement 32 octets (AES-256-GCM).
	// Toujours une référence ${VAR} dans les fichiers, jamais une valeur.
	EncryptionKey string `yaml:"encryption_key"`
	Argon2        Argon2 `yaml:"argon2"`
}

// Argon2 porte le coût du hachage de mot de passe.
type Argon2 struct {
	MemoryKiB  uint32 `yaml:"memory_kib"`
	Iterations uint32 `yaml:"iterations"`
	Threads    uint8  `yaml:"threads"`
}

// DecodedEncryptionKey décode et valide la clé de chiffrement.
func (s Security) DecodedEncryptionKey() ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(s.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("security.encryption_key n'est pas du base64 valide: %w", err)
	}
	if len(key) != encryptionKeyBytes {
		return nil, fmt.Errorf(
			"security.encryption_key doit faire %d octets, reçu %d",
			encryptionKeyBytes, len(key),
		)
	}
	return key, nil
}
