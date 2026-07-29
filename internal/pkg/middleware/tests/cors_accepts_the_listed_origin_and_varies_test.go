package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestCORSAcceptsTheListedOriginAndVaries: the listed origin passes, and the
// response carries `Vary: Origin`.
//
// Two reasons to exist:
//
//   - A blanket refusal would protect just as well and serve no purpose. The
//     test must prove the guard DISCRIMINATES, not that it blocks.
//   - Without `Vary: Origin`, a shared cache — proxy, CDN — can serve one origin
//     the authorised response of another. The guard would then be bypassed by
//     the cache, without anyone touching the code.
func TestCORSAcceptsTheListedOriginAndVaries(t *testing.T) {
	t.Parallel()

	const origin = "https://app.example.com"

	req := get(t)
	req.Header.Set("Origin", origin)
	rec := call(middleware.CORS([]string{origin}), req, okHandler(nil))

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Errorf("Allow-Origin = %q, want %q", got, origin)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin — otherwise a shared cache replays the authorisation", got)
	}
}
