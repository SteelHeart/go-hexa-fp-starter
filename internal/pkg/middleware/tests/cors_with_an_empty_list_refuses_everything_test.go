package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestCORSWithAnEmptyListRefusesEverything: empty list means total refusal.
//
// Deny by default, and the case is far from theoretical: it is the configuration
// of a service just deployed without `http.allowed_origins` filled in yet. If an
// empty list authorised everything, the most exposed moment in a service's life
// — right after going live, before anyone has reviewed its configuration —
// would also be the least protected.
func TestCORSWithAnEmptyListRefusesEverything(t *testing.T) {
	t.Parallel()

	for name, allowed := range map[string][]string{
		"nil list":   nil,
		"empty list": {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := get(t)
			req.Header.Set("Origin", "https://app.example.com")
			rec := call(middleware.CORS(allowed), req, okHandler(nil))

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("no origin authorised in configuration, yet granted: %q", got)
			}
		})
	}
}
