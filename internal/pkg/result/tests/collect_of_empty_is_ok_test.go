package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestCollectOfEmptyIsOk: an empty list collects into a SUCCESS carrying an
// empty list.
//
// Not obvious: since the zero value of a Result is an Err, returning an error by
// accident would be easy. Yet "nothing to process" is not a failure, and an
// empty page of results is not an outage.
func TestCollectOfEmptyIsOk(t *testing.T) {
	t.Parallel()

	empty := result.Collect([]result.Result[int, failure]{})
	if !empty.IsOk() {
		t.Fatalf("an empty list must return a success, got %q", causeOf(empty))
	}
	if got := valueOf(empty); len(got) != 0 {
		t.Errorf("values = %v, want an empty list", got)
	}

	missing := result.Collect[int, failure](nil)
	if !missing.IsOk() {
		t.Error("a nil list must behave like an empty list")
	}
}
