package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestFetchLimitAsksForOneExtraRow: we ask for ONE row more than needed.
//
// That is what tells whether a next page exists without running a `COUNT(*)`. On
// a large table that COUNT costs a full scan on every page — more expensive than
// the page itself, for a piece of information only needed as a boolean.
func TestFetchLimitAsksForOneExtraRow(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 20)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if got := req.FetchLimit(); got != 21 {
		t.Errorf("FetchLimit = %d, want 21 (limit + 1)", got)
	}
}
