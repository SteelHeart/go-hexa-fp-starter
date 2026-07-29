package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestCollectStopsAtFirstError: Collect turns a list of Results into a Result of
// list, and stops at the first error.
//
// "All or nothing" is the right contract here: returning a partial list would
// force every caller to decide what to do with the missing items, and the first
// one forgetting to would treat an incomplete batch as a complete one.
func TestCollectStopsAtFirstError(t *testing.T) {
	t.Parallel()

	all := result.Collect([]result.Result[int, failure]{okInt(1), okInt(2), okInt(3)})
	if !all.IsOk() {
		t.Fatalf("a list of successes must return a success, got %q", causeOf(all))
	}
	if got := valueOf(all); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("collected values = %v, want [1 2 3]", got)
	}

	partial := result.Collect([]result.Result[int, failure]{
		okInt(1), errInt("second refused"), errInt("third refused"),
	})
	if partial.IsOk() {
		t.Fatal("a single error must fail the whole batch")
	}
	if causeOf(partial) != "second refused" {
		t.Errorf("error = %q, want the FIRST one met", causeOf(partial))
	}
}
