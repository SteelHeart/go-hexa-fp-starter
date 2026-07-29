package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestSomeOfZeroIsNotNone: `Some("")` is PRESENT, and distinct from None.
//
// This is the distinction a nil pointer cannot make and that Option exists to
// provide: "field never filled in" and "field deliberately emptied" are two
// different decisions. Conflating them brings back a default value where someone
// had explicitly asked for emptiness.
func TestSomeOfZeroIsNotNone(t *testing.T) {
	t.Parallel()

	empty := fp.Some("")
	if !empty.IsSome() {
		t.Fatal("Some(\"\") must be present")
	}
	if got := empty.ValueOr("fallback"); got != "" {
		t.Errorf("ValueOr = %q: the fallback overwrote a DELIBERATE empty value", got)
	}

	zero := fp.Some(0)
	if !zero.IsSome() || zero.ValueOr(42) != 0 {
		t.Error("Some(0) must be present and return 0, not the fallback")
	}
}
