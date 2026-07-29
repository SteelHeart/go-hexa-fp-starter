package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestZeroValueIsNone: the zero value of an Option is None.
//
// Same property as for Result, and for the same reason: a forgotten Option field
// must mean "absent", never "present and empty". If the zero value were a Some
// carrying the zero of T, a user with no date of birth would appear born on the
// 1st of January of year 1.
func TestZeroValueIsNone(t *testing.T) {
	t.Parallel()

	var forgotten fp.Option[string]

	if forgotten.IsSome() {
		t.Fatal("the zero value of an Option must be None")
	}
	if !forgotten.IsNone() {
		t.Error("IsNone must be true on the zero value")
	}
	if _, present := forgotten.Get(); present {
		t.Error("Get must return present=false on the zero value")
	}
	if got := forgotten.ValueOr("fallback"); got != "fallback" {
		t.Errorf("ValueOr = %q, want the fallback", got)
	}
}
