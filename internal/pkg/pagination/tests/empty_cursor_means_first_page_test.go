package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestEmptyCursorMeansFirstPage: an empty cursor means "first page"; that is not
// an error.
//
// `HasAfter` tells "no cursor" apart from "cursor at the zero value". Without
// that boolean, the first page would be indistinguishable from a request
// starting on the 1st of January of year 1 — and the SQL query would carry a
// pointless `WHERE created_at > ...`, hence an index walked for nothing.
func TestEmptyCursorMeansFirstPage(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("", 10)
	if err != nil {
		t.Fatalf("an empty cursor must not be an error: %v", err)
	}
	if req.HasAfter {
		t.Error("HasAfter must be false on the first page")
	}

	next, err := pagination.NewRequest(
		pagination.Cursor{CreatedAt: base(), ID: "a"}.Encode(), 10)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if !next.HasAfter {
		t.Error("HasAfter must be true when a cursor is supplied")
	}
	if next.After.ID != "a" {
		t.Errorf("resumed cursor = %q, want \"a\"", next.After.ID)
	}
}
