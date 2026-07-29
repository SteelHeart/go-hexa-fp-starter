package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestFlatMapObeysMonadLaws checks the three laws that make a pipeline
// rearrangeable.
//
//  1. LEFT IDENTITY: FlatMap(Ok(x), f) == f(x). Wrapping a value only to unwrap
//     it right away must change nothing.
//  2. RIGHT IDENTITY: FlatMap(r, Ok) == r. Adding a step that merely succeeds
//     must change nothing.
//  3. ASSOCIATIVITY: FlatMap(FlatMap(r, f), g) == FlatMap(r, x => FlatMap(f(x), g)).
//     This is THE law that matters day to day: it says the PARENTHESISATION of
//     steps is irrelevant. Extracting three steps of a use case into a dedicated
//     function is therefore safe — without it, that would be a behaviour change
//     disguised as a rearrangement.
func TestFlatMapObeysMonadLaws(t *testing.T) {
	t.Parallel()

	f := func(n int) result.Result[int, failure] {
		if n < 0 {
			return errInt("negative")
		}
		return okInt(double(n))
	}
	g := func(n int) result.Result[int, failure] {
		if n > 100 {
			return errInt("too large")
		}
		return okInt(increment(n))
	}

	// 1. Left identity.
	if valueOf(result.FlatMap(okInt(5), f)) != valueOf(f(5)) {
		t.Error("left identity broken")
	}

	// 2. Right identity, on both branches.
	for _, r := range []result.Result[int, failure]{okInt(5), errInt("refused")} {
		neutral := result.FlatMap(r, result.Ok[int, failure])
		if neutral.IsOk() != r.IsOk() || valueOf(neutral) != valueOf(r) || causeOf(neutral) != causeOf(r) {
			t.Error("right identity broken")
		}
	}

	// 3. Associativity, including on inputs that short-circuit.
	for _, start := range []result.Result[int, failure]{okInt(5), okInt(-1), okInt(80), errInt("refused")} {
		left := result.FlatMap(result.FlatMap(start, f), g)
		right := result.FlatMap(start, func(n int) result.Result[int, failure] {
			return result.FlatMap(f(n), g)
		})
		if left.IsOk() != right.IsOk() ||
			valueOf(left) != valueOf(right) ||
			causeOf(left) != causeOf(right) {
			t.Errorf("associativity broken starting from %v", valueOf(start))
		}
	}
}
