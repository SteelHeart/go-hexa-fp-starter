package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/storage/domain"
)

// TestDistinctNamesGetDistinctKeys: two objects with the same base name but
// different paths must not overwrite each other.
//
// An upload that silently replaces that of another user is data loss AND a
// leak: the second one then reads the file of the first. It is the hashing of
// the FULL name, and not of the base name alone, that prevents it.
func TestDistinctNamesGetDistinctKeys(t *testing.T) {
	t.Parallel()

	left, err := domain.SafeKey("customer-1/invoice.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	right, err := domain.SafeKey("customer-2/invoice.pdf")
	if err != nil {
		t.Fatalf("SafeKey: %v", err)
	}
	if left == right {
		t.Errorf("two distinct names share the key %q", left)
	}
}
