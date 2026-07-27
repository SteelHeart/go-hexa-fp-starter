package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestSafeKeyKeepsAReadableSuffix : le nom de base survit en suffixe.
//
// Sans lui, un incident se diagnostique à l'aveugle : un magasin de dix mille
// empreintes hexadécimales ne dit pas lequel de ces objets pose problème. C'est le
// compromis assumé du hachage — la clé reste sûre, elle reste lisible.
func TestSafeKeyKeepsAReadableSuffix(t *testing.T) {
	t.Parallel()

	key, err := domain.SafeKey("rapport-annuel.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	if !strings.HasSuffix(key.String(), "-rapport-annuel.pdf") {
		t.Errorf("clé = %q, le nom lisible a disparu", key)
	}
}
