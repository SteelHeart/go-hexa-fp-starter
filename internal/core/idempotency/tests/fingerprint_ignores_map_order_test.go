package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintIgnoresMapOrder : deux constructions de la même map doivent
// rendre la même empreinte. C'est ce que garantit encoding/json en ordonnant les
// clés, et c'est la raison pour laquelle Fingerprint passe par JSON plutôt que par
// une représentation de structure.
func TestFingerprintIgnoresMapOrder(t *testing.T) {
	t.Parallel()

	left := map[string]int{"a": 1, "b": 2, "c": 3}
	right := map[string]int{"c": 3, "b": 2, "a": 1}

	if domain.Fingerprint(left) != domain.Fingerprint(right) {
		t.Error("l'ordre d'insertion d'une map ne doit pas changer l'empreinte")
	}
}
