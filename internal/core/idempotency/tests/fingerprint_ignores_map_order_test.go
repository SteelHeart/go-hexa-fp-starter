package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestFingerprintIgnoresMapOrder: two constructions of the same map must return
// the same fingerprint. That is what encoding/json guarantees by ordering the
// keys, and it is the reason why Fingerprint goes through JSON rather than
// through a struct representation.
func TestFingerprintIgnoresMapOrder(t *testing.T) {
	t.Parallel()

	left := map[string]int{"a": 1, "b": 2, "c": 3}
	right := map[string]int{"c": 3, "b": 2, "a": 1}

	if domain.Fingerprint(left) != domain.Fingerprint(right) {
		t.Error("the insertion order of a map must not change the fingerprint")
	}
}
