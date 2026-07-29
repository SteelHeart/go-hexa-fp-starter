package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestHardeningHeadersAreAlwaysPresent: the hardening headers are set on EVERY
// response, HSTS included by default.
//
// Each closes a specific attack, and none is visible when missing:
//
//   - `X-Content-Type-Options: nosniff` — without it, a browser may decide an
//     uploaded file is HTML and run it in the site's context.
//   - `X-Frame-Options: DENY` — prevents clickjacking.
//   - `Content-Security-Policy: default-src 'none'` — a JSON API serves no active
//     resource; anything that would run has been injected.
//   - `Strict-Transport-Security` — protects the FIRST request of a later visit,
//     the one an attacker on the network would hijack before any encrypted
//     exchange.
func TestHardeningHeadersAreAlwaysPresent(t *testing.T) {
	t.Parallel()

	rec := call(middleware.SecurityHeaders(), get(t), okHandler(nil))

	expected := map[string]string{
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"Referrer-Policy":            "strict-origin-when-cross-origin",
		"Cross-Origin-Opener-Policy": "same-origin",
	}
	for header, want := range expected {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("Content-Security-Policy missing")
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("HSTS missing from the DEFAULT constructor: protection must be had without asking")
	}
}
