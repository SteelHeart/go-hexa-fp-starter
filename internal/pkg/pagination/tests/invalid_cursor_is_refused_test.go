package tests

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/pagination"
)

// TestInvalidCursorIsRefused: an unreadable or tampered cursor is REFUSED.
//
// The cursor comes from the client, hence from a stranger. Silently falling back
// to "first page" on corrupted input would produce an infinite loop on the
// client side: it would forever re-request the same first page believing it was
// moving forward.
//
// Every cause returns the same sentinel error, so the HTTP adapter can answer
// 400 without knowing the format details.
func TestInvalidCursorIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"empty":              "",
		"invalid base64":     "!!!not base64!!!",
		"no separator":       base64.RawURLEncoding.EncodeToString([]byte("1234567890")),
		"missing identifier": base64.RawURLEncoding.EncodeToString([]byte("1234567890|")),
		"non-numeric stamp":  base64.RawURLEncoding.EncodeToString([]byte("yesterday|user-42")),
		"everything empty":   base64.RawURLEncoding.EncodeToString([]byte("|")),
	}

	for name, encoded := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := pagination.DecodeCursor(encoded); !errors.Is(err, pagination.ErrInvalidCursor) {
				t.Errorf("DecodeCursor = %v, want ErrInvalidCursor", err)
			}
		})
	}
}
