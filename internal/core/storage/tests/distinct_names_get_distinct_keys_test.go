package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestDistinctNamesGetDistinctKeys : deux objets de même nom de base mais de
// chemins différents ne doivent pas s'écraser.
//
// Un téléversement qui remplace silencieusement celui d'un autre utilisateur est
// une perte de données ET une fuite : le second lit ensuite le fichier du premier.
// C'est le hachage du nom COMPLET, et non du seul nom de base, qui l'empêche.
func TestDistinctNamesGetDistinctKeys(t *testing.T) {
	t.Parallel()

	left, err := domain.SafeKey("client-1/facture.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	right, err := domain.SafeKey("client-2/facture.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	if left == right {
		t.Errorf("deux noms distincts partagent la clé %q", left)
	}
}
