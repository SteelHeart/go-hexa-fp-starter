package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestEmptyPageIsNotAnError: zero results is a valid answer.
//
// A search with no match, a list never filled, a page requested right after the
// last one: none of these is an outage. The only trap would be returning a
// `NextCursor` on an empty page — the client would loop.
func TestEmptyPageIsNotAnError(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 10)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	page := pagination.NewPage([]row{}, req, cursorOf)
	if page.HasMore {
		t.Error("an empty page has no continuation")
	}
	if page.NextCursor != "" {
		t.Errorf("NextCursor = %q, want empty", page.NextCursor)
	}
	if len(page.Items) != 0 {
		t.Errorf("items = %d, want 0", len(page.Items))
	}
}
