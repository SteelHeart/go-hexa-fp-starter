package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestTheBusSurvivesConcurrentSubscriptionAndPublication: subscribing while
// publishing does not corrupt the table.
//
// # The defect this test catches
//
// Publishing while holding the READ lock at the same time as a consumer
// subscribes under the WRITE lock is the real scenario of start-up: the modules
// mount while the dispatcher is already running. A read of the map without a
// copy — `handlers := b.handlers[env.Type]` instead of a copy — leaves the
// slice shared with a concurrent `append`, and Go then gives NO guarantee at
// all.
//
// ⚠️ This test only has its full force under `-race`, run in CI (F005: `-race`
// requires CGO, absent from the reference machine). Without it, it exercises
// the lock but does not prove the absence of a race.
func TestTheBusSurvivesConcurrentSubscriptionAndPublication(t *testing.T) {
	t.Parallel()

	const rounds = 50

	bus := messaging.NewInproc(quietLogger())
	var delivered atomic.Int64

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for range rounds {
			bus.Subscribe("user.registered.v1", func(context.Context, messaging.Envelope) error {
				delivered.Add(1)
				return nil
			})
		}
	}()
	go func() {
		defer wg.Done()
		for range rounds {
			if err := bus.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
				t.Errorf("concurrent publication failed: %v", err)
			}
		}
	}()
	wg.Wait()

	// The NUMBER of deliveries depends on the interleaving, so we do not assert
	// it: a test that froze that number would be flaky for a good reason. What
	// is asserted is that once everybody has subscribed, everybody receives.
	before := delivered.Load()
	if err := bus.Publish(context.Background(), envelope("user.registered.v1")); err != nil {
		t.Fatalf("final publication failed: %v", err)
	}
	if got := delivered.Load() - before; got != rounds {
		t.Errorf("%d consumers served after the race, want %d — "+
			"a subscription was lost", got, rounds)
	}
}
