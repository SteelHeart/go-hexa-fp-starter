package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestRetryAbandonsImmediatelyOnShutdown: shutdown is not negotiable.
//
// # The defect this test catches
//
// A `time.Sleep` in place of the `select` on `ctx.Done()` would make the wait
// insensitive to cancellation. At the shutdown of the worker, every publication
// in flight would hold its backoff to the end; the orchestrator, for its part,
// does not wait — it sends SIGKILL after its grace period.
//
// The exact consequence: a message published to the broker but NEVER marked in
// the outbox. It is the only case of this starter that produces a duplicate at
// the consumer, and it would be born at every deployment.
//
// The test measures the DURATION: a ten-second backoff must hand back control
// immediately. There is no other way to tell the two writings apart.
func TestRetryAbandonsImmediatelyOnShutdown(t *testing.T) {
	t.Parallel()

	const backoff = 10 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The cancellation leaves AFTER the first attempt: it is the WAIT between
	// two attempts that is checked, not the refusal to enter the loop.
	//
	// sync.Once rather than a busy wait on a counter: the publisher and the
	// canceller would run on different goroutines, and the shared counter would
	// itself be the race this file claims to hunt.
	var once sync.Once
	always := func(context.Context, messaging.Envelope) error {
		once.Do(cancel)
		return errors.New("broker unreachable")
	}

	publish := messaging.WithRetry(always, 5, backoff)

	started := time.Now()
	err := publish(ctx, envelope("user.registered.v1"))
	elapsed := time.Since(started)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("returned error = %v, want context.Canceled", err)
	}
	if elapsed >= backoff {
		t.Errorf("the publication held its backoff (%s) despite the cancellation — "+
			"the worker would be killed midway, and a published message would stay unmarked", elapsed)
	}
}
