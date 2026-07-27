package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// aesKeyLen impose AES-256, et rien d'autre.
//
// `aes.NewCipher` accepte 16, 24 ou 32 octets : une clé de 16 octets produit
// silencieusement de l'AES-128. Le repli est invisible — tout chiffre et déchiffre
// normalement — alors que la garantie annoncée par ce paquet, et par la
// documentation qui s'appuie dessus, est AES-256.
//
// C'est un fail-open : on croit avoir 256 bits, on en a 128. Deny par défaut exige
// de refuser plutôt que de dégrader.
const aesKeyLen = 32

// ErrInvalidKey signale une clé de chiffrement de mauvaise longueur.
var ErrInvalidKey = errors.New("clé de chiffrement invalide")

// Cipher chiffre et déchiffre des données au repos en AES-256-GCM.
type Cipher struct{ aead cipher.AEAD }

// NewCipher construit un chiffreur à partir d'une clé de 32 octets.
//
// Toute autre longueur est REFUSÉE, y compris celles qu'AES accepte : voir aesKeyLen.
func NewCipher(key []byte) (Cipher, error) {
	if len(key) != aesKeyLen {
		return Cipher{}, fmt.Errorf(
			"%w: %d octets fournis, %d attendus (AES-256)", ErrInvalidKey, len(key), aesKeyLen)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Cipher{}, fmt.Errorf("clé AES invalide: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return Cipher{}, fmt.Errorf("initialisation GCM: %w", err)
	}
	return Cipher{aead: aead}, nil
}

// Encrypt chiffre et authentifie. Le nonce est aléatoire et préfixé au message :
// réutiliser un nonce avec GCM révèle le contenu, il n'est donc jamais dérivé.
func (c Cipher) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("génération du nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt déchiffre et vérifie l'authenticité.
func (c Cipher) Decrypt(encoded string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("chiffré illisible: %w", err)
	}
	size := c.aead.NonceSize()
	if len(raw) < size {
		return nil, errors.New("chiffré trop court")
	}
	plaintext, err := c.aead.Open(nil, raw[:size], raw[size:], nil)
	if err != nil {
		// Message volontairement muet : distinguer « mauvaise clé » de
		// « message altéré » aiderait un attaquant.
		return nil, errors.New("déchiffrement refusé")
	}
	return plaintext, nil
}
