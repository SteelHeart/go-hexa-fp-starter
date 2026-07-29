package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestChainShortCircuitsAtFirstError: Chain stops at the FIRST error and runs no
// later step.
//
// This is the mandated shape for writing a use case. The short circuit is not an
// optimisation: the steps of a pipeline assume the previous ones succeeded. A
// "validate the address" step running after "parse the request" failed would
// work on a zero value, and produce a second error masking the real one.
func TestChainShortCircuitsAtFirstError(t *testing.T) {
	t.Parallel()

	var executed []string

	step := func(name string, out result.Result[int, failure]) func(int) result.Result[int, failure] {
		return func(int) result.Result[int, failure] {
			executed = append(executed, name)
			return out
		}
	}

	out := result.Chain(okInt(1),
		step("first", okInt(2)),
		step("second", errInt("refused")),
		step("third", okInt(4)),
	)

	if out.IsOk() {
		t.Fatal("a failing step must fail the whole chain")
	}
	if causeOf(out) != "refused" {
		t.Errorf("error = %q, want the failing step's one", causeOf(out))
	}
	if len(executed) != 2 {
		t.Errorf("executed steps = %v, want only the first two", executed)
	}
}
