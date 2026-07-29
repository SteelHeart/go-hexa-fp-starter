package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestPageDetectsMoreWithoutLeakingTheExtraRow: the extra row serves to detect
// the continuation, it is NEVER returned.
//
// This is the easiest defect to write here: forgetting the truncation would
// return twenty-one rows for a limit of twenty. The client would show one item
// too many per page, and find it again at the top of the next one.
func TestPageDetectsMoreWithoutLeakingTheExtraRow(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 3)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	// FetchLimit is 4: the database returns one row more than the limit.
	page := pagination.NewPage(rows(4), req, cursorOf)

	if !page.HasMore {
		t.Error("HasMore must be true when the extra row exists")
	}
	if len(page.Items) != 3 {
		t.Fatalf("items returned = %d, want 3 — the witness row leaked", len(page.Items))
	}
	if page.Items[2].ID != "c" {
		t.Errorf("last item = %q, want \"c\"", page.Items[2].ID)
	}
	if page.NextCursor == "" {
		t.Error("a next page exists: NextCursor must be set")
	}
}
