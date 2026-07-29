package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
)

// TestRetryGivesUpAndKeepsTheLastCause: the retry is BOUNDED, and it reports.
//
// # The two defects this test catches
//
//  1. An unbounded retry would block the dispatcher on a single message. The
//     backoff of the outbox — minutes, surviving a restart — is the real
//     recovery policy; this one only absorbs an outage of the order of a
//     second. Confusing the two turns a broker breakdown into a stop of the
//     dispatching: NO event goes out any more, not only the failing one.
//  2. An error replaced by a generic message. The dispatcher logs what is
//     handed back to it; if it receives "publication failed" without the cause,
//     the breakdown is diagnosed by guesswork.
func TestRetryGivesUpAndKeepsTheLastCause(t *testing.T) {
	t.Parallel()

	const attempts = 3
	cause := errors.New("broker unreachable")

	var calls int
	always := func(context.Context, messaging.Envelope) error {
		calls++
		return cause
	}

	publish := messaging.WithRetry(always, attempts, time.Millisecond)
	err := publish(context.Background(), envelope("user.registered.v1"))

	if err == nil {
		t.Fatal("an always failing publisher returned nil — the event would be marked as published")
	}
	if !errors.Is(err, cause) {
		t.Errorf("the original cause is lost: %v", err)
	}
	if calls != attempts {
		t.Errorf("publisher called %d times, want exactly %d — "+
			"the retry must stay bounded", calls, attempts)
	}
}
