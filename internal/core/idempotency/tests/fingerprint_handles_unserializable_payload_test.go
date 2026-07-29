package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintHandlesUnserializablePayload: a non-serialisable payload is a
// programming defect. The contract is to return a usable fingerprint all the
// same rather than propagate an error on a defensive path.
func TestFingerprintHandlesUnserializablePayload(t *testing.T) {
	t.Parallel()

	type withChannel struct {
		Ch chan int
	}
	first := domain.Fingerprint(withChannel{Ch: nil})
	if first == "" {
		t.Fatal("empty fingerprint for a non-serialisable payload")
	}
	if second := domain.Fingerprint(withChannel{Ch: nil}); second != first {
		t.Errorf("unstable fallback fingerprint: %q then %q", first, second)
	}
}
