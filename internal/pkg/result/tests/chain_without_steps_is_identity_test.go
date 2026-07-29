package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestChainWithoutStepsIsIdentity: a chain with no step returns its input as is,
// on both branches.
//
// An edge case that looks theoretical and is not: a step list built dynamically
// — filtered by a feature flag, say — may be empty. It must then be neutral, not
// turn a success into something else.
func TestChainWithoutStepsIsIdentity(t *testing.T) {
	t.Parallel()

	for _, start := range []result.Result[int, failure]{okInt(7), errInt("refused")} {
		out := result.Chain(start)
		if out.IsOk() != start.IsOk() ||
			valueOf(out) != valueOf(start) ||
			causeOf(out) != causeOf(start) {
			t.Error("an empty chain must be neutral")
		}
	}
}
