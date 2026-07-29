package tests

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestMaxBodyRefusesAnOversizedPayload: a declared oversize never reaches the
// handler, and the client is told why.
//
// Without a bound, a client grows the server's memory at will: open a request
// and keep sending forever. This is not a sophisticated attack, it is `curl`
// with a large enough file — and the server dies of memory exhaustion, not of an
// error one could diagnose.
//
// ⚠️ This test asserted, until #141, that *the read fails*. That was true and
// insufficient: a failed read is not an answer. Downstream, huma turns it into
// **500 “cannot read request body”** — a server-fault status for a request that
// simply exceeded a documented limit, which sends the operator hunting for a
// defect that is not there.
//
// The bound now refuses BEFORE the handler, with 413, and names the configured
// limit. What is asserted is therefore the ANSWER, not the symptom.
func TestMaxBodyRefusesAnOversizedPayload(t *testing.T) {
	t.Parallel()

	const limit = 64

	reached := false
	reader := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		reached = true
		_, _ = io.ReadAll(r.Body)
	})

	recorder := call(middleware.MaxBody(limit), post(t, strings.Repeat("a", limit*4)), reader)

	if reached {
		t.Error("an oversized body reached the handler: it must be refused before")
	}
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "limit=64") {
		t.Errorf("the answer must name the CONFIGURED limit, got %q", recorder.Body.String())
	}
}

// TestMaxBodyStillCapsAnUndeclaredBody: the memory bound holds without
// Content-Length.
//
// The 413 above relies on a declared length, and a client can omit it — chunked
// transfer does exactly that. If that were the only mechanism, the bound would
// be trivially bypassed by not declaring a size, which is the hole a
// length-based check invites.
//
// `http.MaxBytesReader` is what actually caps memory, and it is kept underneath
// for this case. The status is worse here — the read fails, and the framework
// reports it — but the property that matters, *the process cannot be made to
// buffer without limit*, is the one being asserted.
func TestMaxBodyStillCapsAnUndeclaredBody(t *testing.T) {
	t.Parallel()

	const limit = 64

	var readErr error
	reader := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	})

	request := post(t, strings.Repeat("a", limit*4))
	// A client that does not declare its size. -1 is what net/http uses for an
	// unknown length, and it is what a chunked request produces.
	request.ContentLength = -1

	call(middleware.MaxBody(limit), request, reader)

	if readErr == nil {
		t.Error("an undeclared body past the bound must still make the read fail")
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
