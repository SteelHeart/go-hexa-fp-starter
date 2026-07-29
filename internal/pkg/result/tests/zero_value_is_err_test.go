package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestZeroValueIsErr is the most important test of the primitive.
//
// The zero value of a Result is an Err. That is "deny by default", all the way
// into the type system: a forgotten Result — an uninitialised field, the return
// of a branch believed unreachable — FAILS, instead of silently succeeding while
// carrying the zero value of T.
//
// Were that property to fall, an unassigned `var r Result[User, Error]` would
// look like a successful registration carrying an empty user.
func TestZeroValueIsErr(t *testing.T) {
	t.Parallel()

	var forgotten result.Result[int, failure]

	if forgotten.IsOk() {
		t.Fatal("the zero value of a Result must be an error, never a success")
	}
	if !forgotten.IsErr() {
		t.Error("IsErr must be true on the zero value")
	}
	if _, _, ok := forgotten.Get(); ok {
		t.Error("Get must return ok=false on the zero value")
	}
	if got := forgotten.ValueOr(42); got != 42 {
		t.Errorf("ValueOr = %d, want the fallback 42", got)
	}
}
