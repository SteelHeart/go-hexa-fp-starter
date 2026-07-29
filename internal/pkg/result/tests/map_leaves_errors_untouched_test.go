package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestMapLeavesErrorsUntouched: a failed Result crosses Map without the
// transformation being called.
//
// This is the short circuit that makes a pipeline readable: later steps do not
// have to check whether earlier ones succeeded. Were the function called anyway,
// every step would have to guard itself and the whole gain would vanish.
func TestMapLeavesErrorsUntouched(t *testing.T) {
	t.Parallel()

	called := false
	transform := func(n int) string {
		called = true
		return toText(n)
	}

	out := result.Map(errInt("refused"), transform)

	if called {
		t.Error("the transformation must NOT be called on a failed Result")
	}
	if out.IsOk() {
		t.Error("Map on an error must return an error")
	}
	if causeOf(out) != "refused" {
		t.Errorf("the error must pass through unchanged, got %q", causeOf(out))
	}
}
