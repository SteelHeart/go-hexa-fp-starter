package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/userregistration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/notification"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// Tests of the NON EXPORTED identifiers of the dispatcher's composition root.
//
// `{package}/internal_test.go` and not `{package}/tests/`: `rules/tests.md`
// reserves the latter for the public API of a package, and Go FORBIDS
// importing a `main` package. A black-box test of this binary is therefore
// impossible — that is a constraint of the language, not a preference.
//
// What is EXERCISED here: the wiring decisions, the ones that belong to no
// module and that no architecture rule watches. The rest — the replay, the
// trace, the routing — is exercised in the packages concerned.

// modulesConfig forges the configuration of two modules.
func modulesConfig(notifOn, idemOn bool) config.Config {
	return config.Config{Modules: config.Modules{
		notification.Name: {Enabled: notifOn, Driver: "log"},
		idempotency.Name:  {Enabled: idemOn, Driver: "memory"},
	}}
}

// testLogger captures what the wiring writes.
func testLogger() (*bytes.Buffer, *slog.Logger) {
	buffer := &bytes.Buffer{}
	return buffer, slog.New(slog.NewTextHandler(buffer, nil))
}

// testBroker mounts the in-memory relay.
func testBroker(t *testing.T, logger *slog.Logger) messaging.Broker {
	t.Helper()

	broker, err := messaging.New(config.Messaging{Driver: "inproc"}, logger)
	if err != nil {
		t.Fatalf("test relay: %v", err)
	}
	t.Cleanup(func() { _ = broker.Close() })
	return broker
}

// TestConsumingWithoutIdempotencyIsRefused guards against replaying effects.
//
// # Why a refusal and not a warning
//
// Every transport here is "at least once": the same envelope arrives twice as
// soon as an acknowledgement is lost, which is commonplace. Subscribing
// without idempotency would therefore send the welcome email twice, and the
// day a consumer charges a card, it will be the charge.
//
// A warning at start-up is read once then never. The refusal, on the other
// hand, names what has to change and is repaired in one line of configuration.
func TestConsumingWithoutIdempotencyIsRefused(t *testing.T) {
	t.Parallel()

	_, logger := testLogger()
	err := subscribe(modulesConfig(true, false), testBroker(t, logger), logger)

	if !errors.Is(err, errIdempotencyRequired) {
		t.Fatalf("want errIdempotencyRequired, got %v", err)
	}
}

// TestADisabledNotificationSaysSoRatherThanStayingSilent names the consequence.
//
// A module that is off is a configured state, not a failure: the start-up must
// not fail. But SILENCE would be worse than inaction — a dispatcher publishing
// to nobody looks in every respect like a dispatcher that works: the messages
// are marked published, the outbox drains, the metrics are green, and no
// effect takes place.
//
// The log must therefore name the CONSEQUENCE, not merely the state.
func TestADisabledNotificationSaysSoRatherThanStayingSilent(t *testing.T) {
	t.Parallel()

	buffer, logger := testLogger()
	// Idempotency is disabled too: with no subscription, it is not required.
	// The test therefore ALSO observes that the refusal does not fire when it
	// should not — a guard that is too broad is a guard people end up removing.
	if err := subscribe(modulesConfig(false, false), testBroker(t, logger), logger); err != nil {
		t.Fatalf("a module that is off must not fail the start-up: %v", err)
	}

	trace := buffer.String()
	if !strings.Contains(trace, "user.registered.v1") {
		t.Fatalf("the log must name the event with no consumer: %s", trace)
	}
	if !strings.Contains(trace, "nobody reacts to them") {
		t.Fatalf("the log must name the consequence, not merely the state: %s", trace)
	}
}

// TestTheWelcomeHandlerReadsThePublishedContract guards the published language.
//
// The payload is decoded into `contract.UserRegisteredV1` — primitive types,
// readable by a consumer written in another language. Decoding into a type of
// the `user_registration` domain would recreate the coupling the published
// language serves to avoid.
//
// The test goes through a REAL envelope, serialised the way the producer
// serialises it: that is the only way to catch a JSON field name that would
// have drifted on one side without the other.
func TestTheWelcomeHandlerReadsThePublishedContract(t *testing.T) {
	t.Parallel()

	_, logger := testLogger()
	notif, err := notification.New(
		config.Module{Enabled: true, Driver: "log"}, notification.Deps{Logger: logger})
	if err != nil {
		t.Fatalf("notification module: %v", err)
	}

	payload, err := json.Marshal(contract.UserRegisteredV1{
		UserID: "account-1", Email: "Alice@Example.COM",
	})
	if err != nil {
		t.Fatalf("serialisation: %v", err)
	}

	err = welcome(notif)(context.Background(), messaging.Envelope{
		ID: "evt-1", Type: contract.EventUserRegisteredV1, Payload: payload,
	})
	if err != nil {
		t.Fatalf("the welcome email must go out: %v", err)
	}
}

// TestAnUnreadablePayloadIsReportedNotSwallowed forbids silent forgetting.
//
// An unreadable payload travels up as an error. Swallowing it would
// acknowledge the envelope, hence lose the event for good — and it is up to
// the dispatcher to give up after N attempts, not up to this handler to decide
// on its own to forget.
//
// An invalid address follows the same path: it comes from the producer, so it
// signals a defect there, not a transport failure.
func TestAnUnreadablePayloadIsReportedNotSwallowed(t *testing.T) {
	t.Parallel()

	_, logger := testLogger()
	notif, err := notification.New(
		config.Module{Enabled: true, Driver: "log"}, notification.Deps{Logger: logger})
	if err != nil {
		t.Fatalf("notification module: %v", err)
	}

	emptyAddress, err := json.Marshal(contract.UserRegisteredV1{UserID: "account-1"})
	if err != nil {
		t.Fatalf("serialisation: %v", err)
	}

	cases := map[string][]byte{
		"unreadable payload": []byte("{not json"),
		"empty payload":      nil,
		"missing address":    emptyAddress,
	}
	for name, payload := range cases {
		err := welcome(notif)(context.Background(), messaging.Envelope{
			ID: "evt-1", Type: contract.EventUserRegisteredV1, Payload: payload,
		})
		if err == nil {
			t.Errorf("%s: the error must travel up, not be swallowed", name)
		}
	}
}
