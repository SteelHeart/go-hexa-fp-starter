package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestEachEncryptionUsesAFreshNonce : deux chiffrements du MÊME clair donnent deux
// messages différents.
//
// Avec GCM, réutiliser un nonce sur deux messages ne fait pas qu'affaiblir le
// chiffrement : cela révèle le XOR des deux clairs, et compromet la clé
// d'authentification. C'est la faute la plus grave possible sur ce mode, et elle
// est indétectable à l'usage — les deux messages se déchiffrent parfaitement.
//
// C'est pour cela que le nonce est ALÉATOIRE et jamais dérivé d'un identifiant, si
// tentant soit-il de le rendre reproductible.
func TestEachEncryptionUsesAFreshNonce(t *testing.T) {
	t.Parallel()

	cipher, err := security.NewCipher(cleAES())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	const clair = "une donnée personnelle"
	vus := map[string]struct{}{}
	for range 5 {
		chiffre, err := cipher.Encrypt([]byte(clair))
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		if _, deja := vus[chiffre]; deja {
			t.Fatal("deux chiffrements du même clair ont produit le même message : nonce réutilisé")
		}
		vus[chiffre] = struct{}{}
	}
}
