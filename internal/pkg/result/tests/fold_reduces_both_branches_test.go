package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestFoldReducesBothBranches: Fold is the canonical way out of a Result inside
// a primary adapter.
//
// It forces BOTH branches to be handled: returning an HTTP response having
// forgotten the error case is impossible, since both functions are required at
// the call site. Exactly one branch must be taken, never both.
func TestFoldReducesBothBranches(t *testing.T) {
	t.Parallel()

	var taken []string
	onOk := func(n int) string {
		taken = append(taken, "ok")
		return "success:" + toText(n)
	}
	onErr := func(e failure) string {
		taken = append(taken, "err")
		return "failure:" + string(e)
	}

	if got := result.Fold(okInt(7), onOk, onErr); got != "success:7" {
		t.Errorf("Fold on a success = %q", got)
	}
	if got := result.Fold(errInt("refused"), onOk, onErr); got != "failure:refused" {
		t.Errorf("Fold on an error = %q", got)
	}
	if len(taken) != 2 {
		t.Errorf("branches taken = %v, want exactly one per call", taken)
	}
}
