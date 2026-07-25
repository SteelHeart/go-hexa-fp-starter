// Package security fournit le hachage de mot de passe et le chiffrement au repos.
//
// Rien ici ne connaît le métier : ce sont des primitives, branchées sur des
// ports par le composition root.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Paramètres du format d'encodage Argon2id.
const (
	argon2SaltLen = 16
	argon2KeyLen  = 32
	argon2Version = 19
	argon2Parts   = 6
)

// ErrInvalidHash signale un condensé illisible : format inconnu, version
// inattendue, ou encodage corrompu.
var ErrInvalidHash = errors.New("condensé de mot de passe invalide")

// Argon2Params porte le coût du hachage.
//
// Augmenter Memory est ce qui protège réellement contre le calcul massivement
// parallèle : augmenter uniquement Iterations coûte autant à l'attaquant sur GPU
// qu'au serveur.
type Argon2Params struct {
	MemoryKiB  uint32
	Iterations uint32
	Threads    uint8
}

// Hasher hache et vérifie des mots de passe avec Argon2id.
type Hasher struct{ params Argon2Params }

// NewHasher construit un hacheur.
func NewHasher(params Argon2Params) Hasher { return Hasher{params: params} }

// Hash produit un condensé auto-décrit : le format embarque la version et les
// paramètres, ce qui permet d'augmenter le coût sans invalider les condensés
// existants.
//
//	$argon2id$v=19$m=65536,t=3,p=4$<sel b64>$<clé b64>
func (h Hasher) Hash(plain string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("génération du sel: %w", err)
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

// Verify compare un mot de passe en clair à un condensé, en temps constant.
//
// Une erreur de format et un mot de passe incorrect sont deux choses
// différentes : la première est un défaut de données, la seconde un cas nominal.
func (h Hasher) Verify(plain, encoded string) (bool, error) {
	params, salt, key, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	candidate := argon2.IDKey(
		[]byte(plain), salt,
		params.Iterations, params.MemoryKiB, params.Threads, uint32(len(key)),
	)
	return subtle.ConstantTimeCompare(key, candidate) == 1, nil
}

// NeedsRehash indique si un condensé a été produit avec un coût inférieur au
// coût courant. À appeler après une vérification réussie pour remonter le coût
// de façon transparente.
func (h Hasher) NeedsRehash(encoded string) bool {
	params, _, _, err := decodeHash(encoded)
	if err != nil {
		return true
	}
	return params.MemoryKiB < h.params.MemoryKiB ||
		params.Iterations < h.params.Iterations
}

func decodeHash(encoded string) (Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != argon2Parts || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, fmt.Errorf("%w: préfixe", ErrInvalidHash)
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2Version {
		return Argon2Params{}, nil, nil, fmt.Errorf("%w: version", ErrInvalidHash)
	}
	var params Argon2Params
	if _, err := fmt.Sscanf(
		parts[3], "m=%d,t=%d,p=%d",
		&params.MemoryKiB, &params.Iterations, &params.Threads,
	); err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("%w: paramètres", ErrInvalidHash)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("%w: sel", ErrInvalidHash)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, fmt.Errorf("%w: clé", ErrInvalidHash)
	}
	return params, salt, key, nil
}

// Cipher chiffre et déchiffre des données au repos en AES-256-GCM.
type Cipher struct{ aead cipher.AEAD }

// NewCipher construit un chiffreur à partir d'une clé de 32 octets.
func NewCipher(key []byte) (Cipher, error) {
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

// BlindIndex produit un index déterministe permettant de rechercher une valeur
// chiffrée sans la déchiffrer.
//
// ⚠️ Déterministe, donc il fuit l'égalité : deux valeurs identiques ont le même
// index. Acceptable pour une recherche exacte sur un champ à forte entropie,
// jamais pour un champ à faible cardinalité.
func BlindIndex(key, value []byte) string {
	mac := sha256.Sum256(append(append([]byte(nil), key...), value...))
	return base64.RawURLEncoding.EncodeToString(mac[:])
}
