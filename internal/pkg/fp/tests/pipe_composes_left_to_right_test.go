package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestPipeComposesLeftToRight: Pipe2 and Pipe3 compose in READING order.
//
// The order is not a matter of taste. Mathematical composition is written right
// to left — `g o f` applies f first — and that is the opposite of what a reader
// assumes. The repository picks reading order, and checks it, because a reversed
// composition produces a plausible result: `toText(double(3))` and
// `double(toText(3))` do not both compile, but two transformations of the same
// type do.
func TestPipeComposesLeftToRight(t *testing.T) {
	t.Parallel()

	// double THEN increment: 3 -> 6 -> 7. The reverse order would give 8.
	two := fp.Pipe2(double, increment)
	if got := two(3); got != 7 {
		t.Errorf("Pipe2 = %d, want 7 (double then increment)", got)
	}

	// double THEN increment THEN toText: 3 -> 6 -> 7 -> "7".
	three := fp.Pipe3(double, increment, toText)
	if got := three(3); got != "7" {
		t.Errorf("Pipe3 = %q, want \"7\"", got)
	}
}
