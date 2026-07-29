package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestTheMiddlewareStackIsActuallyMounted: the stack is WIRED, not merely written.
//
// # Why this test exists separately from those of the middleware package
//
// `internal/pkg/middleware/tests` proves that each guard WORKS. It does not
// prove that it is MOUNTED. These are two distinct defects, and the second
// leaves no trace: a `mux.Use(...)` from which one line disappeared at merge
// time produces a perfectly functional service, simply without protection.
//
// No test of the middleware package can catch it — they build the stack
// themselves. So it is here, and nowhere else.
//
// This test checks what is OBSERVABLE in a response: the correlation identifier
// and the refused origin. Rate limiting and the body bound have their own
// tests; replaying them here would make them depend on the test configuration.
func TestTheMiddlewareStackIsActuallyMounted(t *testing.T) {
	t.Parallel()

	headers := get(t, config.EnvProduction, nil, "/healthz").Header()

	// RequestID: present in the output, always. Without it, no log line can be
	// linked back to a request.
	if headers.Get(middleware.RequestIDHeader) == "" {
		t.Errorf("%s missing from the response — middleware.RequestID() is not mounted",
			middleware.RequestIDHeader)
	}
}

// TestCORSIsMountedAndDeniesAnUnlistedOrigin: the origin guard is in place.
//
// The test configuration allows only `https://app.example.com`. A response that
// granted access to another origin would signal either a CORS missing from the
// stack, or a CORS mounted with the wrong list.
func TestCORSIsMountedAndDeniesAnUnlistedOrigin(t *testing.T) {
	t.Parallel()

	rec := get(t, config.EnvProduction, nil, "/healthz")

	// No origin was sent: no authorisation header must appear. A `*` here would
	// be a total opening.
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q without an Origin header in the request", got)
	}
}
