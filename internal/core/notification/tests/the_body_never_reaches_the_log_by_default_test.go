package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
)

// TestTheBodyNeverReachesTheLogByDefault guards the safe default.
//
// # What this test protects
//
// A notification body regularly carries a secret: confirmation link, reset
// token, one-time code. These are BEARER credentials — whoever reads them can
// use them, without a password and without leaving a trace.
//
// The application log is precisely what one exports the most willingly to a
// third-party collector. A body logged by default would therefore turn every log
// leak into an account takeover.
func TestTheBodyNeverReachesTheLogByDefault(t *testing.T) {
	t.Parallel()

	mod, logs := newModule(t, nil)
	if err := mod.Send(context.Background(), message(t)); err != nil {
		t.Fatalf("send: %v", err)
	}

	trace := logs.text()
	if trace == "" {
		t.Fatal("the driver wrote nothing: the test observes nothing")
	}
	if strings.Contains(trace, secretInBody) {
		t.Fatalf("the body leaks in the log: %s", trace)
	}
	if strings.Contains(trace, recipient) {
		t.Fatalf("the address in clear leaks in the log: %s", trace)
	}

	// What MUST be there: enough to diagnose without revealing anything.
	for _, want := range []string{"notification", subject, "body_bytes"} {
		if !strings.Contains(trace, want) {
			t.Errorf("the log must carry %q: %s", want, trace)
		}
	}
}

// TestTheBodyIsLoggedOnlyWhenAsked establishes that the option is READ.
//
// A silently ignored configuration option is the defect that cost issue #93: the
// server started, mounted the driver, and said nothing.
//
// The test also checks that the warning travels in the SAME record as the body.
// In two separate lines, the first gets lost in the sorting — and whoever reads
// the secret never reads the caveat.
func TestTheBodyIsLoggedOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	mod, logs := newModule(t, map[string]any{notification.OptionBody: notification.BodyLogged})
	if err := mod.Send(context.Background(), message(t)); err != nil {
		t.Fatalf("send: %v", err)
	}

	trace := logs.text()
	if !strings.Contains(trace, secretInBody) {
		t.Fatalf("the body should have been logged: %s", trace)
	}
	if !strings.Contains(trace, "development only") {
		t.Fatalf("the warning must accompany the body: %s", trace)
	}

	// The address stays masked EVEN when the body is written: the two decisions
	// are independent, and conflating them would drop the second by enabling the
	// first.
	if strings.Contains(trace, recipient) {
		t.Fatalf("the address in clear leaks although only the body was asked for: %s", trace)
	}
}

// TestAnUnknownBodyOptionRefusesStartup: deny by default all the way into the
// option.
//
// The fallback to `masked` would be tempting though — it is "safe", since it
// keeps the body silent. But someone writing `body: logging` would believe they
// asked for the body, would not see it, and would search elsewhere for an hour.
func TestAnUnknownBodyOptionRefusesStartup(t *testing.T) {
	t.Parallel()

	logs := newLogCapture()
	for _, value := range []any{"logging", "true", "verbose", ""} {
		_, err := notification.New(
			configModule(map[string]any{notification.OptionBody: value}),
			notification.Deps{Logger: logs.logger},
		)
		if err == nil {
			t.Errorf("body=%q: startup must be refused", value)
		}
	}
}
