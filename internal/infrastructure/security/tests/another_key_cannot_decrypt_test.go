package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestAnotherKeyCannotDecrypt : une autre clé ne déchiffre pas.
//
// Évident sur le papier, décisif en exploitation : c'est ce qui rend une rotation de
// clé visible. Si une clé erronée déchiffrait quand même, une rotation ratée ne se
// remarquerait qu'au moment où les données seraient devenues illisibles pour tout le
// monde — bien après le déploiement fautif.
func TestAnotherKeyCannotDecrypt(t *testing.T) {
	t.Parallel()

	original, err := security.NewCipher(cleAES())
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	chiffre, err := original.Encrypt([]byte("une donnée personnelle"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	autreCle := cleAES()
	autreCle[0] ^= 0xFF
	autre, err := security.NewCipher(autreCle)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	if _, err := autre.Decrypt(chiffre); err == nil {
		t.Error("une autre clé ne doit PAS pouvoir déchiffrer")
	}
}
