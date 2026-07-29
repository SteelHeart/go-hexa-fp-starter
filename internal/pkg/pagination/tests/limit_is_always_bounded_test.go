package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestLimitIsAlwaysBounded: an unbounded page is a denial of service, offered.
//
// `?limit=1000000` on a table of several million rows is enough to exhaust the
// process memory — with no particular authentication, no tool, in a single
// request. The ceiling is therefore not a comfort setting.
//
// Zero and negative values fall back to the default rather than being refused: a
// missing limit is the nominal case of a first call.
func TestLimitIsAlwaysBounded(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		requested int
		want      int
	}{
		"missing":          {requested: 0, want: pagination.DefaultLimit},
		"negative":         {requested: -10, want: pagination.DefaultLimit},
		"reasonable":       {requested: 50, want: 50},
		"at the ceiling":   {requested: pagination.MaxLimit, want: pagination.MaxLimit},
		"past the ceiling": {requested: 1_000_000, want: pagination.MaxLimit},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req, err := pagination.NewRequest("", tc.requested)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if req.Limit != tc.want {
				t.Errorf("limit = %d, want %d", req.Limit, tc.want)
			}
		})
	}
}
