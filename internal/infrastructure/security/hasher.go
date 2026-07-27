package security

import (
	"crypto/rand"
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

// Bornes du condensé décodé.
//
// Un condensé est censé venir de notre propre base, mais rien ne le garantit : une
// reprise de données, une colonne mal migrée, un magasin externe suffisent à y
// glisser autre chose. Sans borne, la longueur de clé annoncée pilote directement
// une allocation dans argon2.IDKey — un « condensé » forgé annonçant quatre
// gigaoctets ferait tomber le processus à la première vérification.
//
// Les bornes couvrent large pour ne pas invalider un condensé légitime produit
// avec une autre longueur de clé, tout en fermant le cas absurde.
const (
	argon2MinKeyLen = 16
	argon2MaxKeyLen = 64
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
	decoded, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	// La conversion est sûre : decodeHash a déjà refusé toute longueur hors bornes.
	keyLen := uint32(len(decoded.key)) //nolint:gosec // borné par decodeHash
	candidate := argon2.IDKey(
		[]byte(plain), decoded.salt,
		decoded.params.Iterations, decoded.params.MemoryKiB, decoded.params.Threads, keyLen,
	)
	return subtle.ConstantTimeCompare(decoded.key, candidate) == 1, nil
}

// NeedsRehash indique si un condensé a été produit avec un coût inférieur au
// coût courant. À appeler après une vérification réussie pour remonter le coût
// de façon transparente.
func (h Hasher) NeedsRehash(encoded string) bool {
	decoded, err := decodeHash(encoded)
	if err != nil {
		// Un condensé illisible mérite d'être refait, pas d'être conservé.
		return true
	}
	return decoded.params.MemoryKiB < h.params.MemoryKiB ||
		decoded.params.Iterations < h.params.Iterations
}

// decodedHash porte les trois morceaux d'un condensé.
//
// Regroupés en type plutôt qu'en trois valeurs de retour : ils n'ont de sens
// qu'ensemble — vérifier une clé avec le sel d'un autre condensé ne veut rien dire —
// et trois retours de même forme (`[]byte`, `[]byte`) s'inversent sans que le
// compilateur bronche.
type decodedHash struct {
	params Argon2Params
	salt   []byte
	key    []byte
}

func decodeHash(encoded string) (decodedHash, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != argon2Parts || parts[1] != "argon2id" {
		return decodedHash{}, fmt.Errorf("%w: préfixe", ErrInvalidHash)
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2Version {
		return decodedHash{}, fmt.Errorf("%w: version", ErrInvalidHash)
	}
	var params Argon2Params
	if _, err := fmt.Sscanf(
		parts[3], "m=%d,t=%d,p=%d",
		&params.MemoryKiB, &params.Iterations, &params.Threads,
	); err != nil {
		return decodedHash{}, fmt.Errorf("%w: paramètres", ErrInvalidHash)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return decodedHash{}, fmt.Errorf("%w: sel", ErrInvalidHash)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return decodedHash{}, fmt.Errorf("%w: clé", ErrInvalidHash)
	}
	if len(key) < argon2MinKeyLen || len(key) > argon2MaxKeyLen {
		return decodedHash{}, fmt.Errorf("%w: longueur de clé (%d octets)", ErrInvalidHash, len(key))
	}
	return decodedHash{params: params, salt: salt, key: key}, nil
}
