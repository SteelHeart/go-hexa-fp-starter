package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestTheInprocBusDeliversToEverySubscriber: several consumers, one event.
//
// # The defect this test catches
//
// A `map[string]Handler` table instead of `map[string][]Handler` — which is
// what both network relays write — silently replaces the previous consumer. The
// second module that subscribes to `user.registered.v1` unsubscribes the first,
// and nobody sees it: each module has its test, each test passes on its own.
//
// The test ALSO checks that the event does not reach another type: a bus that
// broadcasts to everybody works in development, and makes billing run on a
// registration event in production.
func TestTheInprocBusDeliversToEverySubscriber(t *testing.T) {
	t.Parallel()

	bus := messaging.NewInproc(quietLogger())

	var first, second, other int
	bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		first++
		return nil
	})
	bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
		second++
		return nil
	})
	bus.Subscribe("invoice.issued.v1", func(context.Context, messaging.Envelope) error {
		other++
		return nil
	})

	if err := bus.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("publication failed: %v", err)
	}

	if first != 1 || second != 1 {
		t.Errorf("consumers called %d and %d times, want 1 and 1 — "+
			"a subscription must never replace another", first, second)
	}
	if other != 0 {
		t.Errorf("a consumer of ANOTHER type was called %d times", other)
	}

	// Then the OTHER type. Without this second send, the test would stay green
	// on a bus that would deliver ONLY to the first registered type: "nothing
	// reached `other`" is demonstrated just as well by correct routing as by
	// dead routing. One must prove the route exists before proving it is
	// watertight.
	if err := bus.Publish(context.Background(), envelope("invoice.issued.v1")); err != nil {
		t.Fatalf("publication of the second type failed: %v", err)
	}
	if other != 1 {
		t.Errorf("the consumer of invoice.issued.v1 was called %d times, want 1", other)
	}
	if first != 1 || second != 1 {
		t.Errorf("the consumers of user.registered.v1 received an event of ANOTHER type "+
			"(%d and %d) — billing would run on a registration", first, second)
	}
}
