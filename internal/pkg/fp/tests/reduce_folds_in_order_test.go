package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestReduceFoldsInOrder: Reduce folds left to right.
//
// The order is invisible on an addition and decisive on everything else. A
// concatenation folded backwards returns a reversed string — a result that looks
// correct until someone reads it.
func TestReduceFoldsInOrder(t *testing.T) {
	t.Parallel()

	sum := fp.Reduce([]int{1, 2, 3, 4}, 0, func(acc, n int) int { return acc + n })
	if sum != 10 {
		t.Errorf("sum = %d, want 10", sum)
	}

	concat := fp.Reduce([]string{"a", "b", "c"}, "", func(acc, s string) string { return acc + s })
	if concat != "abc" {
		t.Errorf("concatenation = %q, want \"abc\" — the fold goes left to right", concat)
	}
}
