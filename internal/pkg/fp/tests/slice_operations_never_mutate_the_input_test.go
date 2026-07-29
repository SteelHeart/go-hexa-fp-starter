package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestSliceOperationsNeverMutateTheInput is the central guarantee of these three
// functions.
//
// In Go, a slice shares its backing array: an implementation writing into
// `items` would modify the CALLER's slice, at a distance and without a trace.
// That is the most painful class of defects to diagnose, because the culprit is
// a function that looks pure.
func TestSliceOperationsNeverMutateTheInput(t *testing.T) {
	t.Parallel()

	source := []int{1, 2, 3, 4}
	witness := []int{1, 2, 3, 4}

	_ = fp.Map(source, double)
	_ = fp.Filter(source, even)
	_ = fp.Reduce(source, 0, func(acc, n int) int { return acc + n })
	_ = fp.Find(source, even)

	for i := range witness {
		if source[i] != witness[i] {
			t.Fatalf("the input was modified: %v, want %v", source, witness)
		}
	}
}
