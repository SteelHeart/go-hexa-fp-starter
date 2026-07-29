package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestCursorIsURLSafe: a cursor travels in a query string.
//
// A `+`, a `/` or an `=` in it would be re-encoded by a client, a cache or a
// proxy — and the cursor would come back altered. URL-safe base64 without
// padding avoids all three. The defect would only show in production, on some
// clients only, which makes it an excellent candidate for an automated test.
func TestCursorIsURLSafe(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"a", "user-42", "01J8ZQ9V3K4M5N6P7R8S9T0V1W", "e#?&=/+"} {
		encoded := pagination.Cursor{CreatedAt: base(), ID: id}.Encode()
		if strings.ContainsAny(encoded, "+/=") {
			t.Errorf("cursor %q holds a character re-encoded in a URL", encoded)
		}
	}
}
