package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestRateLimiterIgnoresAForgeableHeader: `X-Forwarded-For` does not change the
// client's identity.
//
// # Why this is a security decision, not an oversight
//
// Taking `X-Forwarded-For` as the identity would make throttling TRIVIALLY
// bypassable: sending a different value on every request is enough to get a
// fresh quota each time. The guard would look like it works — it would indeed
// refuse past the quota if nobody cheated — while protecting against nothing at
// all from anyone who tries.
//
// The header is only usable once rewritten by a trusted proxy. As long as that
// proxy is not described in the configuration, it does not exist, and only
// `RemoteAddr` counts.
func TestRateLimiterIgnoresAForgeableHeader(t *testing.T) {
	t.Parallel()

	const burst = 2
	limiter := middleware.NewRateLimiter(0.001, burst, time.Minute).Middleware()

	for range burst {
		_ = requestFrom(t, limiter, "10.0.0.9:1111")
	}

	// Same real address, header forged on every call: the quota must stay
	// exhausted.
	for attempt := range 3 {
		req := get(t)
		req.RemoteAddr = "10.0.0.9:1111"
		req.Header.Set("X-Forwarded-For", "203.0.113."+string(rune('1'+attempt)))

		if code := call(limiter, req, okHandler(nil)).Code; code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d = %d, want 429 — X-Forwarded-For must not grant a fresh quota",
				attempt+1, code)
		}
	}
}
