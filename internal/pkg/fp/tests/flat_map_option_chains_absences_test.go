package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestFlatMapOptionChainsAbsences: chaining lookups that may find nothing must
// not produce an Option of Option.
//
// That is the whole difference with MapOption: looking up the employer of a user
// we are not sure exists returns "no employer", not "maybe a maybe".
func TestFlatMapOptionChainsAbsences(t *testing.T) {
	t.Parallel()

	positiveOrNothing := func(n int) fp.Option[int] {
		if n <= 0 {
			return fp.None[int]()
		}
		return fp.Some(n)
	}

	if got := fp.FlatMapOption(fp.Some(5), positiveOrNothing); !got.IsSome() {
		t.Error("a valid value must pass through")
	}
	if got := fp.FlatMapOption(fp.Some(-5), positiveOrNothing); got.IsSome() {
		t.Error("a value refused by f must give None")
	}

	called := false
	missing := fp.FlatMapOption(fp.None[int](), func(n int) fp.Option[int] {
		called = true
		return fp.Some(n)
	})
	if called {
		t.Error("f must NOT be called on an empty Option")
	}
	if missing.IsSome() {
		t.Error("None must pass through unchanged")
	}
}
