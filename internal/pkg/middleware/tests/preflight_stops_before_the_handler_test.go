package tests

import (
	"net/http"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestPreflightStopsBeforeTheHandler: an OPTIONS stops at the middleware.
//
// Letting the preflight reach the handler would make it run the business logic
// of a request that is NOT the real one — the browser has not sent anything
// useful yet. On a write route, that would amount to running the operation
// twice, once of them without a body.
func TestPreflightStopsBeforeTheHandler(t *testing.T) {
	t.Parallel()

	const origin = "https://app.example.com"

	reached := false
	rec := call(middleware.CORS([]string{origin}), preflight(t, origin), okHandler(&reached))

	if reached {
		t.Error("the handler was reached by an OPTIONS preflight")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("the preflight must announce the allowed methods")
	}
}
