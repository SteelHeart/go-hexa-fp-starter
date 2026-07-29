package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestTraverseStopsAtFirstError: Traverse applies a fallible function to every
// item and stops at the first error — WITHOUT calling the function on the
// remaining items.
//
// Actually stopping matters as much as the result: if the function produces an
// effect — writing to a database, calling a service — carrying on after an error
// would leave a partial state nobody asked for.
func TestTraverseStopsAtFirstError(t *testing.T) {
	t.Parallel()

	var seen []int
	validate := func(n int) result.Result[string, failure] {
		seen = append(seen, n)
		if n < 0 {
			return result.Err[string, failure]("negative value")
		}
		return result.Ok[string, failure](toText(n))
	}

	all := result.Traverse([]int{1, 2, 3}, validate)
	if !all.IsOk() {
		t.Fatalf("no error expected, got %q", causeOf(all))
	}
	if got := valueOf(all); len(got) != 3 || got[2] != "3" {
		t.Errorf("values = %v, want [1 2 3] as text", got)
	}

	seen = nil
	partial := result.Traverse([]int{1, -2, 3}, validate)
	if partial.IsOk() {
		t.Fatal("an invalid item must fail the whole traversal")
	}
	if len(seen) != 2 {
		t.Errorf("items traversed = %v, want the stop at the second", seen)
	}
}
