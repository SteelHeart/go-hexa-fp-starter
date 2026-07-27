package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
)

// TestCipherRefusesAKeyOfWrongLength : une clé qui n'est pas de 32 octets refuse la
// construction.
//
// Le refus doit avoir lieu au DÉMARRAGE, pas à la première donnée chiffrée. Une clé
// mal dimensionnée vient presque toujours d'une variable d'environnement tronquée ou
// mal encodée — le genre de défaut qu'on veut découvrir au déploiement, pas la nuit
// suivante.
func TestCipherRefusesAKeyOfWrongLength(t *testing.T) {
	t.Parallel()

	for name, taille := range map[string]int{
		"vide": 0, "trop courte": 16, "presque": 31, "trop longue": 64,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := security.NewCipher(make([]byte, taille)); err == nil {
				t.Errorf("une clé de %d octets doit être refusée", taille)
			}
		})
	}

	if _, err := security.NewCipher(cleAES()); err != nil {
		t.Errorf("une clé de 32 octets doit être acceptée: %v", err)
	}
}
