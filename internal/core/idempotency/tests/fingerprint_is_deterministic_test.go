package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintIsDeterministic: without determinism, the same request
// replayed would produce a different fingerprint and would be seen as a
// conflict. The module would then refuse legitimate replays — exactly the
// opposite of its role.
func TestFingerprintIsDeterministic(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"email": "a@example.com", "amount": 4200, "active": true}
	first := domain.Fingerprint(payload)

	for range 20 {
		if got := domain.Fingerprint(payload); got != first {
			t.Fatalf("unstable fingerprint: %q then %q", first, got)
		}
	}
}
