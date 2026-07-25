package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintNeverEmpty : une empreinte vide vaudrait « pas d'empreinte », et
// domain.ErrIncomplete refuserait une requête pourtant valide.
func TestFingerprintNeverEmpty(t *testing.T) {
	t.Parallel()

	payloads := []any{nil, "", 0, false, map[string]any{}, []int{}, struct{}{}}
	for _, payload := range payloads {
		if got := domain.Fingerprint(payload); got == "" {
			t.Errorf("empreinte vide pour %#v", payload)
		}
	}
}
