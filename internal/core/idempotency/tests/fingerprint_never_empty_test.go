package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintNeverEmpty: an empty fingerprint would mean "no fingerprint",
// and domain.ErrIncomplete would refuse an otherwise valid request.
func TestFingerprintNeverEmpty(t *testing.T) {
	t.Parallel()

	payloads := []any{nil, "", 0, false, map[string]any{}, []int{}, struct{}{}}
	for _, payload := range payloads {
		if got := domain.Fingerprint(payload); got == "" {
			t.Errorf("empty fingerprint for %#v", payload)
		}
	}
}
