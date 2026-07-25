package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestEncryptThenDecryptRoundTrips : le chemin nominal du chiffrement au repos.
//
// Une donnée chiffrée qu'on ne sait plus déchiffrer est une donnée PERDUE, sans
// message d'erreur au moment où on la perd. C'est le seul test qui l'empêche.
func TestEncryptThenDecryptRoundTrips(t *testing.T) {
	t.Parallel()

	cipher, err := security.NewCipher(cleAES())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	for _, clair := range []string{
		"", "a", "une donnée personnelle", "accents éàü et symboles ✓",
	} {
		chiffre, err := cipher.Encrypt([]byte(clair))
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", clair, err)
		}
		dechiffre, err := cipher.Decrypt(chiffre)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", clair, err)
		}
		if string(dechiffre) != clair {
			t.Errorf("aller-retour = %q, attendu %q", dechiffre, clair)
		}
	}
}
