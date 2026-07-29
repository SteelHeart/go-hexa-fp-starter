package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestRateLimiterIsolatesEachClient: a client that saturates does not affect the
// others.
//
// # The defect this test catches
//
// A limiter counting GLOBALLY instead of PER CLIENT turns the protection into a
// weapon: a single attacker is then enough to put everyone on 429. The service
// stays up, answers, logs no error — and nobody can use it.
//
// That defect never shows in development, where there is only one client.
func TestRateLimiterIsolatesEachClient(t *testing.T) {
	t.Parallel()

	const burst = 2
	limiter := middleware.NewRateLimiter(0.001, burst, time.Minute).Middleware()

	// The first client exhausts its quota.
	for i := range burst {
		if code := requestFrom(t, limiter, "10.0.0.1:1234"); code != http.StatusOK {
			t.Fatalf("request %d of the first client = %d, want 200", i+1, code)
		}
	}
	if code := requestFrom(t, limiter, "10.0.0.1:1234"); code != http.StatusTooManyRequests {
		t.Errorf("past the quota = %d, want 429", code)
	}

	// ANOTHER client must keep its own.
	if code := requestFrom(t, limiter, "10.0.0.2:5678"); code != http.StatusOK {
		t.Errorf("second client = %d, want 200 — the quota is PER CLIENT", code)
	}
}

// requestFrom sends a request from a given address.
func requestFrom(t *testing.T, mw func(http.Handler) http.Handler, remoteAddr string) int {
	t.Helper()

	req := get(t)
	req.RemoteAddr = remoteAddr
	return call(mw, req, okHandler(nil)).Code
}
