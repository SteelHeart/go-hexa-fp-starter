package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestMapOptionSkipsAbsentValues: the transformation is NOT called on an empty
// Option.
//
// That is what allows chaining transformations without testing presence at every
// step. Were the function called on absence, it would receive the zero value of T
// and produce a result that looks valid.
func TestMapOptionSkipsAbsentValues(t *testing.T) {
	t.Parallel()

	called := false
	out := fp.MapOption(fp.None[int](), func(n int) string {
		called = true
		return toText(n)
	})

	if called {
		t.Error("the transformation must NOT be called on an empty Option")
	}
	if out.IsSome() {
		t.Error("MapOption on None must return None")
	}
}
