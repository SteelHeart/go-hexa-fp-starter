package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestEmptySlicesAreNotASpecialCase: an empty OR nil slice passes through
// without panicking and returns an empty slice, never nil.
//
// Returning nil would force every caller to tell "empty" from "nil" — a
// distinction Go makes almost invisible and that resurfaces at serialisation:
// `null` instead of `[]` in a JSON response breaks typed clients.
func TestEmptySlicesAreNotASpecialCase(t *testing.T) {
	t.Parallel()

	for name, source := range map[string][]int{"empty": {}, "nil": nil} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := fp.Map(source, double); got == nil {
				t.Error("Map must return an empty slice, never nil")
			} else if len(got) != 0 {
				t.Errorf("Map = %v, want empty", got)
			}

			if got := fp.Filter(source, even); got == nil {
				t.Error("Filter must return an empty slice, never nil")
			} else if len(got) != 0 {
				t.Errorf("Filter = %v, want empty", got)
			}

			if got := fp.Reduce(source, 100, func(acc, n int) int { return acc + n }); got != 100 {
				t.Errorf("Reduce = %d, want the initial value untouched", got)
			}

			if got := fp.Find(source, even); got.IsSome() {
				t.Error("Find in an empty slice must return None")
			}
		})
	}
}
