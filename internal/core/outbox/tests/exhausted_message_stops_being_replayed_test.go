package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestExhaustedMessageStopsBeingReplayed: after N tries, we stop.
//
// A poisoned message — payload unreadable by the consumer, recipient gone —
// will never be published. Replaying it forever would drown the logs and take
// up every batch, delaying the healthy messages behind it.
//
// It therefore goes to `failed` and is no longer picked up. It is NOT deleted:
// it is the only trace of what has not been published, and the event deserves
// human intervention — hence the distinct report.
func TestExhaustedMessageStopsBeingReplayed(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	handle := func(context.Context, domain.Message) error {
		return errors.New("unreadable payload")
	}

	// The test policy allows three attempts in total.
	policy := testPolicy()
	// The message has already failed twice: this try is the third.
	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, claimOnce(pending("m-1", 2)), handle), policy)

	if _, err := dispatcher.DrainOnce(context.Background()); err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}

	attempt := observed.lastAttempt(t)
	if attempt.Attempts != policy.Retry.MaxAttempts {
		t.Errorf("attempts = %d, want %d", attempt.Attempts, policy.Retry.MaxAttempts)
	}
	if attempt.Status != domain.StatusFailed {
		t.Errorf("status = %q, want %q", attempt.Status, domain.StatusFailed)
	}
	if attempt.Reason == "" {
		t.Error("the cause must be kept: without it, the trace is unusable")
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventExhausted {
		t.Errorf("event = %q, want %q", got, domain.EventExhausted)
	}
}
