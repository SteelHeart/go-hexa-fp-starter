package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestChainAppliesStepsInOrder: steps apply in the declared order, and each one
// receives the previous one's output.
//
// Order is the only thing a reader infers from the shape of the code. Were it
// not guaranteed, reading a use case would no longer tell what it does.
func TestChainAppliesStepsInOrder(t *testing.T) {
	t.Parallel()

	// The first and third steps are identical, and that is the heart of the
	// test: the same function applied at two different positions must produce
	// two different results (2 then 6). Deduplicating what gocritic believes to
	// be a repetition would destroy the demonstration — nothing would be left
	// proving that position matters.
	out := result.Chain(okInt(1),
		func(n int) result.Result[int, failure] { return okInt(double(n)) },    // 2
		func(n int) result.Result[int, failure] { return okInt(increment(n)) }, // 3
		//nolint:gocritic // the repetition IS the demonstration: same step, two positions
		func(n int) result.Result[int, failure] { return okInt(double(n)) }, // 6
	)

	if !out.IsOk() {
		t.Fatalf("chain failed: %q", causeOf(out))
	}
	if got := valueOf(out); got != 6 {
		t.Errorf("value = %d, want 6 — steps are not chained in order", got)
	}
}
