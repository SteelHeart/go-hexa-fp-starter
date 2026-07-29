package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestMapObeysFunctorLaws checks the two laws that make refactoring safe.
//
//  1. IDENTITY: Map(r, x => x) == r. A transformation that transforms nothing
//     must change nothing — otherwise inserting a neutral step into a pipeline
//     would change its result.
//  2. COMPOSITION: Map(Map(r, f), g) == Map(r, x => g(f(x))). This is the law
//     that allows MERGING two steps into one, or extracting a third, without
//     breaking anything. Without it, any regrouping of steps would be a gamble.
func TestMapObeysFunctorLaws(t *testing.T) {
	t.Parallel()

	identity := func(n int) int { return n }

	for _, r := range []result.Result[int, failure]{okInt(21), errInt("refused")} {
		applied := result.Map(r, identity)
		if applied.IsOk() != r.IsOk() ||
			valueOf(applied) != valueOf(r) ||
			causeOf(applied) != causeOf(r) {
			t.Error("identity law broken")
		}
	}

	start := okInt(10)
	inTwoSteps := result.Map(result.Map(start, double), increment)
	inOneStep := result.Map(start, func(n int) int { return increment(double(n)) })

	if valueOf(inTwoSteps) != valueOf(inOneStep) {
		t.Errorf("composition law broken: %d != %d",
			valueOf(inTwoSteps), valueOf(inOneStep))
	}
}
