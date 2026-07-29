package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
)

// TestADisabledModuleRefusesItNeverSwallows guards the worst possible silence.
//
// # The defect this test prevents
//
// A disabled notification module returning `nil` would make every message count
// as sent. Nothing would fail, no log would say anything, and the defect would
// only show at the first customer claiming they never received their email —
// weeks later, without any trace to go back on.
//
// It is the costliest form of a defect: the one that looks exactly like success.
func TestADisabledModuleRefusesItNeverSwallows(t *testing.T) {
	t.Parallel()

	mod, err := notification.New(config.Module{Enabled: false}, notification.Deps{})
	if err != nil {
		t.Fatalf("a disabled module must mount: %v", err)
	}

	if err := mod.Send(context.Background(), message(t)); !errors.Is(err, notification.ErrDisabled) {
		t.Fatalf("want ErrDisabled, got %v", err)
	}
}

// TestUnknownDriverRefusesStartup: deny by default all the way into the factory.
//
// `smtp`, `mailjet` and `ses` appear in the catalogue of intentions and are NOT
// built. The refusal must be plain: a silent fallback to `log` would run a
// production that writes its emails to a log instead of sending them — and
// nothing would report it.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	logs := newLogCapture()
	for _, driver := range []string{"smtp", "mailjet", "ses", "sendgrid", "journal"} {
		_, err := notification.New(
			config.Module{Enabled: true, Driver: driver},
			notification.Deps{Logger: logs.logger},
		)
		if err == nil {
			t.Errorf("driver %q is not built: it must refuse startup", driver)
		}
	}
}

// TestMissingLoggerRefusesStartup fails the assembly, not production.
//
// Without this refusal, the first message would produce a nil pointer
// dereference — hence in production, and on an asynchronous path where the
// consumer's panic would be confused with a malformed message.
func TestMissingLoggerRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := notification.New(config.Module{Enabled: true, Driver: "log"}, notification.Deps{})
	if !errors.Is(err, notification.ErrMissingDependency) {
		t.Fatalf("want ErrMissingDependency, got %v", err)
	}
}

// TestEmptyDriverFallsBackToTheDefault: specifying nothing takes `log`.
//
// That is what makes the "`hexa new` then `go run`" promise true on the COMPLETE
// chain — registration, outbox, relay, notification — without an SMTP server nor
// an account at a provider.
func TestEmptyDriverFallsBackToTheDefault(t *testing.T) {
	t.Parallel()

	logs := newLogCapture()
	mod, err := notification.New(config.Module{Enabled: true}, notification.Deps{Logger: logs.logger})
	if err != nil {
		t.Fatalf("with no driver specified, the default must apply: %v", err)
	}
	if err := mod.Send(context.Background(), message(t)); err != nil {
		t.Fatalf("send on the default driver: %v", err)
	}
	if logs.text() == "" {
		t.Fatal("the default driver must be the `log` driver")
	}
}

// TestAMessageNeverPrintsItsBody covers BOTH formatting verbs.
//
// `%v` goes through `String()`, `%#v` through `GoString()`. Covering one leaves
// the other leaking, and `%#v` is precisely what one writes in a debug log —
// hence on the day of an incident, hence on the day the logs go to a third
// party.
//
// ⚠️ The test formats DELIBERATELY rather than calling `String()`: it is the real
// leak path that is exercised. Calling the method would leave the test green if
// someone removed the `Stringer`.
func TestAMessageNeverPrintsItsBody(t *testing.T) {
	t.Parallel()

	msg := message(t)
	for _, rendered := range []string{
		fmt.Sprintf("%v", msg),  //nolint:gocritic // this is the leak path under test
		fmt.Sprintf("%#v", msg), // through GoString — the other half of the mask
		fmt.Sprint(msg),         //nolint:gocritic // likewise, without a verb
	} {
		if strings.Contains(rendered, secretInBody) {
			t.Fatalf("the body leaks in %q", rendered)
		}
		if strings.Contains(rendered, recipient) {
			t.Fatalf("the address in clear leaks in %q", rendered)
		}
		if !strings.Contains(rendered, "***") {
			t.Fatalf("the mask must be visible in %q", rendered)
		}
	}
}
