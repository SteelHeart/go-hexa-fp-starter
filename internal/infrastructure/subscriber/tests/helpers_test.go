// Package tests exercises the wiring of a consumer onto the guarantees of the
// core.
//
// # The idempotency module is the REAL one, not a double
//
// The `memory` driver is the one that runs in development, and its 24 tests
// prove exclusivity under concurrency. Doubling it here would exercise the
// double: the decorator could respect a contract that the real module does not
// keep, and the replay would still get through in production.
//
// What IS doubled is the handler — because what is being measured is precisely
// the number of times it is called.
package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// eventType is the event type used by the tests.
const eventType = "user.registered.v1"

// newGuard mounts the REAL idempotency module on its default driver.
func newGuard(t *testing.T) subscriber.Guard {
	t.Helper()

	mod, err := idempotency.New(
		config.Module{Enabled: true, Driver: "memory", Options: map[string]any{"ttl": "1h"}},
		idempotency.Deps{Now: time.Now},
	)
	if err != nil {
		t.Fatalf("mounting idempotency: %v", err)
	}
	return subscriber.Guard{Reserve: mod.Reserve, Complete: mod.Complete, Release: mod.Release}
}

// envelope forges a transport envelope.
func envelope(id string) messaging.Envelope {
	return messaging.Envelope{
		ID:          id,
		Type:        eventType,
		AggregateID: "account-1",
		Payload:     []byte(`{"user_id":"account-1","email":"alice@example.com"}`),
		OccurredAt:  time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC),
	}
}

// counter retains how many times the handler has been called.
//
// It is the only thing these tests really measure: "did the effect take place,
// and how many times". A concurrency-safe counter, because one of the tests
// replays from several goroutines.
type counter struct {
	mu       sync.Mutex
	calls    int
	failure  error
	contexts []context.Context
}

// handler returns a handler that counts its calls.
func (c *counter) handler() messaging.Handler {
	return func(ctx context.Context, _ messaging.Envelope) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.calls++
		c.contexts = append(c.contexts, ctx)
		return c.failure
	}
}

// total returns the number of observed calls.
func (c *counter) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// lastContext returns the context received on the last call.
func (c *counter) lastContext(t *testing.T) context.Context {
	t.Helper()

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.contexts) == 0 {
		t.Fatal("the handler has never been called: the test observes nothing")
	}
	return c.contexts[len(c.contexts)-1]
}

// failing returns a counter whose handler always fails.
func failing(err error) *counter { return &counter{failure: err} }
