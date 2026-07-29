package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestSafeKeyKeepsAReadableSuffix: the base name survives as a suffix.
//
// Without it, an incident is diagnosed blind: a store of ten thousand
// hexadecimal digests does not say which of those objects is the problem. This
// is the accepted trade-off of the hashing — the key stays safe, it stays
// readable.
func TestSafeKeyKeepsAReadableSuffix(t *testing.T) {
	t.Parallel()

	key, err := domain.SafeKey("annual-report.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	if !strings.HasSuffix(key.String(), "-annual-report.pdf") {
		t.Errorf("key = %q, the readable name has disappeared", key)
	}
}
