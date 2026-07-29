package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestCORSRefusesAnUnlistedOrigin: an unlisted origin gets NO authorisation
// header at all.
//
// # Why this is the most important test of the package
//
// An over-permissive CORS breaks nothing. The page works, the logs are clean, no
// alert fires. It is discovered the day a third-party site reads the
// authenticated responses of a logged-in user — and by then the leak has already
// happened.
//
// That is exactly the profile of a defect a test must guard: invisible until
// exploited. The cases below are the ones an approximate comparison — prefix,
// suffix, "contains" — would let through.
func TestCORSRefusesAnUnlistedOrigin(t *testing.T) {
	t.Parallel()

	allowed := []string{"https://app.example.com"}
	refused := map[string]string{
		"unknown origin":            "https://evil.example.com",
		"misleading suffix":         "https://app.example.com.evil.com",
		"same host in cleartext":    "http://app.example.com",
		"wildcard":                  "*",
		"prefix of the allowed one": "https://app.example.co",
	}

	for name, origin := range refused {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := get(t)
			req.Header.Set("Origin", origin)
			rec := call(middleware.CORS(allowed), req, okHandler(nil))

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("origin %q authorised (%q) while it is not listed", origin, got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
				t.Errorf("origin %q gets Allow-Credentials", origin)
			}
		})
	}
}
