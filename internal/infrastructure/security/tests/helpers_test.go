// Package tests contient les tests en BOÎTE NOIRE des primitives de sécurité :
// ils n'utilisent que l'API publique, exactement comme un adaptateur.
//
// Convention du dépôt (rules/tests.md) : `{paquet}/tests/` pour la boîte noire,
// `{paquet}/internal_test.go` pour les identifiants non exportés. Un fichier par
// test — le nom du fichier dit ce qui est vérifié, sans avoir à l'ouvrir.
//
// # Pourquoi ce paquet mérite plus de tests que les autres
//
// Une erreur de hachage ou de chiffrement ne se manifeste JAMAIS à l'usage : tout
// continue de fonctionner. Un condensé mal formé se vérifie quand même, un message
// chiffré avec un nonce réutilisé se déchiffre quand même. Le défaut n'apparaît que
// le jour où quelqu'un l'exploite — c'est-à-dire trop tard, et sans avertissement.
package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// testParams est un coût VOLONTAIREMENT bas.
//
// Les paramètres de production sont conçus pour être lents : c'est leur rôle. Les
// utiliser ici rendrait la suite inutilisable, et une suite lente finit par ne plus
// être lancée. Le comportement testé — format, bornes, comparaison — ne dépend pas
// du coût.
func testParams() security.Argon2Params {
	return security.Argon2Params{MemoryKiB: 1 << 10, Iterations: 1, Threads: 1}
}

// newHasher construit un hacheur de test.
func newHasher() security.Hasher { return security.NewHasher(testParams()) }

// hash produit un condensé, ou fait échouer le test.
func hash(t *testing.T, plain string) string {
	t.Helper()
	encoded, err := newHasher().Hash(plain)
	if err != nil {
		t.Fatalf("Hash(%q): %v", plain, err)
	}
	return encoded
}

// cléAES rend une clé de 32 octets déterministe.
//
// Fabriquée à l'exécution, jamais écrite en dur : rules/securite.md interdit tout
// secret versionné, et une chaîne de 32 octets dans un fichier est indiscernable
// d'une vraie fuite, pour gitleaks comme pour un lecteur.
func cleAES() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

// clone recopie une tranche, pour altérer sans toucher à l'original.
func clone(src []byte) []byte { return append([]byte(nil), src...) }
