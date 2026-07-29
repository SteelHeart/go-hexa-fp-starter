package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestMapOptionObeysFunctorLaws: identity and composition, for the same reasons
// as on Result.
//
// Without the composition law, merging two successive transformations into one
// would change the result — and that kind of regrouping happens without thinking
// during a code cleanup.
func TestMapOptionObeysFunctorLaws(t *testing.T) {
	t.Parallel()

	identity := func(n int) int { return n }

	for _, o := range []fp.Option[int]{fp.Some(21), fp.None[int]()} {
		applied := fp.MapOption(o, identity)
		gotV, gotP := applied.Get()
		wantV, wantP := o.Get()
		if gotV != wantV || gotP != wantP {
			t.Error("identity law broken")
		}
	}

	start := fp.Some(10)
	inTwo := fp.MapOption(fp.MapOption(start, double), increment)
	inOne := fp.MapOption(start, func(n int) int { return increment(double(n)) })

	if inTwo.ValueOr(-1) != inOne.ValueOr(-1) {
		t.Errorf("composition law broken: %d != %d", inTwo.ValueOr(-1), inOne.ValueOr(-1))
	}
}
