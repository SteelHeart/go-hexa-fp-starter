package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestNextCursorResumesExactlyAfterTheLastItem: the next cursor names the LAST
// item returned, not the witness row.
//
// This is the heart of the pagination contract. Were it to name the extra row —
// the one read but not returned — the next page would start one notch too far
// and that item would NEVER be displayed. The defect is silent: nothing reports
// a missing row between two pages.
func TestNextCursorResumesExactlyAfterTheLastItem(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 3)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	page := pagination.NewPage(rows(4), req, cursorOf)
	last := page.Items[len(page.Items)-1]

	resumed, err := pagination.DecodeCursor(page.NextCursor)
	if err != nil {
		t.Fatalf("the returned cursor must be decodable: %v", err)
	}
	if resumed.ID != last.ID {
		t.Errorf("next cursor = %q, want the last item RETURNED (%q)",
			resumed.ID, last.ID)
	}
	if !resumed.CreatedAt.Equal(last.CreatedAt) {
		t.Errorf("cursor timestamp = %v, want %v", resumed.CreatedAt, last.CreatedAt)
	}
}
