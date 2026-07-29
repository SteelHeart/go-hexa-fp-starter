package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestOrElseRecoversOnlyFromErrors: OrElse replaces an error with a fallback,
// and leaves a success untouched.
//
// It is the only way to recover without leaving the box. Were it to touch
// successes too, it would become a "always replace" — and the fallback would
// overwrite legitimate values without anything reporting it.
func TestOrElseRecoversOnlyFromErrors(t *testing.T) {
	t.Parallel()

	fallback := func(failure) result.Result[int, failure] { return okInt(0) }

	recovered := result.OrElse(errInt("cache unavailable"), fallback)
	if !recovered.IsOk() {
		t.Fatal("OrElse must replace an error with the fallback")
	}
	if valueOf(recovered) != 0 {
		t.Errorf("fallback value = %d, want 0", valueOf(recovered))
	}

	called := false
	untouched := result.OrElse(okInt(7), func(failure) result.Result[int, failure] {
		called = true
		return okInt(0)
	})
	if called {
		t.Error("the fallback must NOT be called on a success")
	}
	if valueOf(untouched) != 7 {
		t.Errorf("value = %d, want 7 unchanged", valueOf(untouched))
	}
}
