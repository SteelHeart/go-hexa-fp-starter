package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestPanickingHandlerDoesNotKillTheDispatcher: a poisoned message must not
// take down the worker.
//
// The publisher is caller code: a nil map access in a consumer is enough to
// panic. Without a net, that panic would climb the goroutine and kill the whole
// process — and, with the `memory` driver, would leave the message claimed
// forever, hence invisible AND never published.
//
// The panic is therefore treated as an ordinary failure: backoff policy,
// abandonment after N tries, and a report that clearly distinguishes it from an
// error returned cleanly.
func TestPanickingHandlerDoesNotKillTheDispatcher(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	handle := func(context.Context, domain.Message) error {
		var absent map[string]string
		// Writing into a nil map is INTENDED: it is the shortest way to provoke a
		// real runtime panic, the one a faulty driver would produce. A literal
		// `panic()` would be less faithful: the dispatcher must survive the panics
		// it did not see coming, not only our own.
		absent["boom"] = "panic" //nolint:govet,staticcheck // panic provoked on purpose, that is the object of the test
		return nil
	}

	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, claimOnce(pending("m-1", 0)), handle), testPolicy())

	// Must not panic: this is the main assertion, and it is implicit.
	count, err := dispatcher.DrainOnce(context.Background())
	if err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}
	if count != 1 {
		t.Errorf("processed messages = %d, want 1", count)
	}

	if len(observed.done) != 0 {
		t.Error("a message whose publication panicked must not be marked processed")
	}
	attempt := observed.lastAttempt(t)
	if attempt.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempt.Attempts)
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventHandlerPanicked {
		t.Errorf("event = %q, want %q", got, domain.EventHandlerPanicked)
	}
}
