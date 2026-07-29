package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// TestRequestIDNeverTrustsTheCaller: a hostile correlation identifier is
// replaced, never propagated.
//
// # The attack path
//
// This identifier ENDS UP IN THE LOGS. A caller slipping a carriage return into
// it therefore injects whole lines into the server's log: fake entries, fake
// severity levels, visual erasure of an inconvenient trace. This is log
// injection, and it targets precisely what serves to investigate afterwards.
//
// An oversized value, for its part, inflates every log line of a request — hence
// storage cost and analysis time.
//
// The middleware reuses the supplied identifier when it is plausible, which is
// useful to follow a call across several services. "Plausible" must therefore be
// checked, not assumed.
func TestRequestIDNeverTrustsTheCaller(t *testing.T) {
	t.Parallel()

	hostile := map[string]string{
		"carriage return": "abc\r\nlevel=ERROR msg=\"fake incident\"",
		"line feed":       "abc\ninjection",
		"oversized":       strings.Repeat("x", 500),
		"empty":           "",
	}

	for name, value := range hostile {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := get(t)
			req.Header.Set(middleware.RequestIDHeader, value)
			got := call(middleware.RequestID(), req, okHandler(nil)).Header().Get(middleware.RequestIDHeader)

			if got == value {
				t.Errorf("hostile identifier propagated as is: %q", got)
			}
			if strings.ContainsAny(got, "\r\n") {
				t.Errorf("the returned identifier holds a line break: %q", got)
			}
			if got == "" {
				t.Error("an identifier must always be returned, even after refusing the received one")
			}
		})
	}
}
