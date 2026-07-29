package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestFindReturnsTheFirstMatch: Find returns the FIRST item satisfying the
// predicate, and None if none does.
//
// "First" is the contract, not an implementation accident: on an ordered list —
// most recent first — returning the last instead of the first would reverse the
// meaning of the search without any type objecting.
//
// Absence returns None rather than a zero value: that is what tells "no result"
// apart from "an empty result".
func TestFindReturnsTheFirstMatch(t *testing.T) {
	t.Parallel()

	found := fp.Find([]int{1, 3, 4, 6, 8}, even)
	if !found.IsSome() {
		t.Fatal("an item satisfies the predicate: Find must find it")
	}
	if got, _ := found.Get(); got != 4 {
		t.Errorf("Find = %d, want 4 — the FIRST even one, not another", got)
	}

	if missing := fp.Find([]int{1, 3, 5}, even); missing.IsSome() {
		t.Error("no item satisfies the predicate: Find must return None")
	}
}
