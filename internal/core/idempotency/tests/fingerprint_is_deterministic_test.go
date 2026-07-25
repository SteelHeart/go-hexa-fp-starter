package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintIsDeterministic : sans déterminisme, la même requête rejouée
// produirait une empreinte différente et serait vue comme un conflit. Le module
// refuserait alors des rejeux légitimes — exactement l'inverse de son rôle.
func TestFingerprintIsDeterministic(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"email": "a@example.com", "montant": 4200, "actif": true}
	first := domain.Fingerprint(payload)

	for range 20 {
		if got := domain.Fingerprint(payload); got != first {
			t.Fatalf("empreinte instable: %q puis %q", first, got)
		}
	}
}
