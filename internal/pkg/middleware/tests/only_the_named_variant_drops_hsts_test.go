package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestOnlyTheNamedVariantDropsHSTS: only the variant that NAMES itself gives up
// HSTS.
//
// # What this test really protects
//
// It used to be `SecurityHeaders(secure bool)`. A control boolean says neither
// what it enables nor what it costs: `SecurityHeaders(false)` in the wrong place
// removed HSTS in production without review seeing it.
//
// The opt-out now carries a name, and this test checks both directions — that
// the default protects, and that the named variant is the ONLY one that does
// not. Checking only one of the two would let the regression that matters
// through.
//
// The opt-out stays legitimate in development: on `http://localhost`, HSTS would
// record in the browser an HTTPS requirement the machine cannot satisfy, and the
// developer would lose access to their own server until clearing the cache.
func TestOnlyTheNamedVariantDropsHSTS(t *testing.T) {
	t.Parallel()

	withHSTS := call(middleware.SecurityHeaders(), get(t), okHandler(nil))
	without := call(middleware.SecurityHeadersWithoutHSTS(), get(t), okHandler(nil))

	if got := withHSTS.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("SecurityHeaders() must set HSTS")
	}
	if got := without.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("SecurityHeadersWithoutHSTS() set HSTS: %q", got)
	}

	// ALL the rest of the hardening must survive the opt-out: we give up HSTS,
	// not protection.
	for _, header := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy"} {
		if got := without.Header().Get(header); got == "" {
			t.Errorf("%s lost by the variant without HSTS — it gives up HSTS only", header)
		}
	}
}
