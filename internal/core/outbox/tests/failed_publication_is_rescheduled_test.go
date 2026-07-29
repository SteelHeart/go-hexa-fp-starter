package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestFailedPublicationIsRescheduled: an unreachable broker is not a loss.
//
// The message stays `pending`, its counter goes up, and its availability date
// is pushed back. Marking an unpublished message `done` would be the silent
// loss that the whole outbox pattern exists to prevent.
func TestFailedPublicationIsRescheduled(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	outage := errors.New("broker unreachable")
	handle := func(context.Context, domain.Message) error { return outage }

	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, claimOnce(pending("m-1", 0)), handle), testPolicy())

	if _, err := dispatcher.DrainOnce(context.Background()); err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}

	if len(observed.done) != 0 {
		t.Error("an unpublished message must NEVER be marked as processed")
	}

	attempt := observed.lastAttempt(t)
	if attempt.Status != domain.StatusPending {
		t.Errorf("status = %q, want %q: the message must stay replayable",
			attempt.Status, domain.StatusPending)
	}
	if attempt.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempt.Attempts)
	}
	if !attempt.AvailableAt.After(fixedNow()) {
		t.Error("the next try must be pushed into the future")
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventRetryScheduled {
		t.Errorf("event = %q, want %q", got, domain.EventRetryScheduled)
	}
}
