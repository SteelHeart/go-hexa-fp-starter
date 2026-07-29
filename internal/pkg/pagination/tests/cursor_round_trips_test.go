package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestCursorRoundTrips: encoding then decoding yields the original cursor.
//
// This is the property the whole pagination depends on: a cursor makes a round
// trip over the network, and if it does not come back identical, the caller
// resumes somewhere other than where it left off — hence skips or repeats rows.
//
// Precision is to the MICROSECOND, deliberately: that is PostgreSQL's for
// `timestamptz`. Keeping nanoseconds in memory would produce a cursor the
// database cannot find again.
func TestCursorRoundTrips(t *testing.T) {
	t.Parallel()

	cases := map[string]pagination.Cursor{
		"ordinary instant": {CreatedAt: base(), ID: "user-42"},
		"epoch":            {CreatedAt: time.UnixMicro(0).UTC(), ID: "x"},
		"long identifier":  {CreatedAt: base(), ID: "01J8ZQ9V3K4M5N6P7R8S9T0V1W"},
		"before epoch":     {CreatedAt: time.UnixMicro(-1_000_000).UTC(), ID: "old"},
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := pagination.DecodeCursor(want.Encode())
			if err != nil {
				t.Fatalf("decoding: %v", err)
			}
			if !got.CreatedAt.Equal(want.CreatedAt) {
				t.Errorf("timestamp = %v, want %v", got.CreatedAt, want.CreatedAt)
			}
			if got.ID != want.ID {
				t.Errorf("identifier = %q, want %q", got.ID, want.ID)
			}
		})
	}
}
