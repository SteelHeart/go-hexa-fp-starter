package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestTapObservesWithoutAltering: Tap and TapErr produce an effect and return
// the Result UNCHANGED.
//
// They are reserved for decorators — tracing, logging, counting. Could they
// modify the Result, an observability decorator would change the behaviour of
// the application it observes, and the defect would be undetectable: nobody
// suspects the tracer.
func TestTapObservesWithoutAltering(t *testing.T) {
	t.Parallel()

	var seen []string

	succeeded := result.Tap(okInt(7), func(n int) { seen = append(seen, "ok:"+toText(n)) })
	succeeded = result.TapErr(succeeded, func(failure) { seen = append(seen, "err") })

	if valueOf(succeeded) != 7 || !succeeded.IsOk() {
		t.Error("Tap must not alter a success")
	}

	failed := result.Tap(errInt("refused"), func(int) { seen = append(seen, "ok") })
	failed = result.TapErr(failed, func(e failure) { seen = append(seen, "err:"+string(e)) })

	if failed.IsOk() || causeOf(failed) != "refused" {
		t.Error("TapErr must not alter an error")
	}

	if len(seen) != 2 || seen[0] != "ok:7" || seen[1] != "err:refused" {
		t.Errorf("observed effects = %v, want one per branch", seen)
	}
}
