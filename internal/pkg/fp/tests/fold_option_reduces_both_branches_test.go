package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestFoldOptionReducesBothBranches: FoldOption forces absence to be handled.
//
// Both functions are required at the call site: leaving an Option having
// forgotten the empty case is impossible. That is the guarantee the compiler
// brings here and that a nil pointer never brings.
func TestFoldOptionReducesBothBranches(t *testing.T) {
	t.Parallel()

	onSome := func(n int) string { return "present:" + toText(n) }
	onNone := func() string { return "absent" }

	if got := fp.FoldOption(fp.Some(7), onSome, onNone); got != "present:7" {
		t.Errorf("FoldOption on Some = %q", got)
	}
	if got := fp.FoldOption(fp.None[int](), onSome, onNone); got != "absent" {
		t.Errorf("FoldOption on None = %q", got)
	}
}
