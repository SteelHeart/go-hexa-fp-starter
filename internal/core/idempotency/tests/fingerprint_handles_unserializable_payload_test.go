package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintHandlesUnserializablePayload : une charge non sérialisable est un
// défaut de programmation. Le contrat est de rendre quand même une empreinte
// utilisable plutôt que de propager une erreur sur un chemin défensif.
func TestFingerprintHandlesUnserializablePayload(t *testing.T) {
	t.Parallel()

	type withChannel struct {
		Ch chan int
	}
	first := domain.Fingerprint(withChannel{Ch: nil})
	if first == "" {
		t.Fatal("empreinte vide pour une charge non sérialisable")
	}
	if second := domain.Fingerprint(withChannel{Ch: nil}); second != first {
		t.Errorf("empreinte de repli instable: %q puis %q", first, second)
	}
}
