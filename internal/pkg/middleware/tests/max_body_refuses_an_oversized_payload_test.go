package tests

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestMaxBodyRefusesAnOversizedPayload: reading stops at the bound.
//
// Without a bound, a client grows the server's memory at will: open a request
// and keep sending forever. This is not a sophisticated attack, it is `curl`
// with a large enough file — and the server dies of memory exhaustion, not of an
// error one could diagnose.
//
// The test ACTUALLY reads the body: the bound only applies on read, so a handler
// that does not read proves nothing.
func TestMaxBodyRefusesAnOversizedPayload(t *testing.T) {
	t.Parallel()

	const limit = 64

	var readErr error
	reader := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	})

	call(middleware.MaxBody(limit), post(t, strings.Repeat("a", limit*4)), reader)

	if readErr == nil {
		t.Error("a body past the bound must make the read fail")
	}
}

// TestMaxBodyLetsAnAcceptablePayloadThrough: under the bound, nothing changes.
//
// A bound that also refused legitimate traffic would be removed at the first
// complaint — and the protection with it.
func TestMaxBodyLetsAnAcceptablePayloadThrough(t *testing.T) {
	t.Parallel()

	const limit = 64
	payload := strings.Repeat("a", limit/2)

	var got string
	reader := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read under the bound failed: %v", err)
		}
		got = string(raw)
	})

	call(middleware.MaxBody(limit), post(t, payload), reader)

	if got != payload {
		t.Errorf("body read = %d bytes, want %d", len(got), len(payload))
	}
}
