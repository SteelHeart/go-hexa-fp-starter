package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestInvalidCursorRejectsTheWholeRequest: an invalid cursor fails the ENTIRE
// request.
//
// The temptation would be to keep the limit and start over from the first page.
// That would be a silent fallback on corrupted input: the client would believe
// it was moving through a list while endlessly re-reading its beginning.
func TestInvalidCursorRejectsTheWholeRequest(t *testing.T) {
	t.Parallel()

	req, err := pagination.NewRequest("!!!not base64!!!", 50)
	if !errors.Is(err, pagination.ErrInvalidCursor) {
		t.Fatalf("error = %v, want ErrInvalidCursor", err)
	}
	if req.Limit != 0 || req.HasAfter {
		t.Errorf("the request must be returned EMPTY on error, got %+v", req)
	}
}
