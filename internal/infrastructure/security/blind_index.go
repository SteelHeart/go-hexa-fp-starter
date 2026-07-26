package security

import (
	"crypto/sha256"
	"encoding/base64"
)

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
