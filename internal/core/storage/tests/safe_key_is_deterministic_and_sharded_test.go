package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestSafeKeyIsDeterministicAndSharded: two calls on the same name give the
// same key, and the key is spread over two levels of directories.
//
// The spreading is not cosmetic: a flat directory with a hundred thousand
// entries becomes impractical on most file systems, and the store slows down
// all at once, late, without anyone understanding why.
func TestSafeKeyIsDeterministicAndSharded(t *testing.T) {
	t.Parallel()

	first, err := domain.SafeKey("invoice-2026-07.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	second, err := domain.SafeKey("invoice-2026-07.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	if first != second {
		t.Errorf("unstable key: %q then %q", first, second)
	}

	parts := strings.Split(first.String(), "/")
	if len(parts) != 3 {
		t.Fatalf("key = %q, want three levels", first)
	}
	if len(parts[0]) != 2 || len(parts[1]) != 2 {
		t.Errorf("spreading = %q/%q, want two levels of 2 characters", parts[0], parts[1])
	}
}
