package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestLastPageHasNoNextCursor: the last page offers no continuation.
//
// Returning a cursor when there is nothing left would loop a client following
// cursors to exhaustion: it would endlessly re-request an empty page that would
// hand it yet another cursor. `HasMore` and `NextCursor` must therefore say the
// same thing, always.
func TestLastPageHasNoNextCursor(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 3)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	cases := map[string]int{"incomplete page": 2, "exactly full page": 3}

	for name, count := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			page := pagination.NewPage(rows(count), req, cursorOf)
			if page.HasMore {
				t.Error("HasMore must be false: no extra row was read")
			}
			if page.NextCursor != "" {
				t.Errorf("NextCursor = %q, want empty on the last page", page.NextCursor)
			}
			if len(page.Items) != count {
				t.Errorf("items = %d, want %d", len(page.Items), count)
			}
		})
	}
}
