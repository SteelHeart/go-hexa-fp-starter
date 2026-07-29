package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestClaimFailureIsReportedAndStops: an unreachable store stops the round
// without losing anything. No message has been claimed, so none is in danger.
func TestClaimFailureIsReportedAndStops(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	outage := errors.New("database unreachable")
	failingClaim := func(context.Context, int) ([]domain.Message, error) { return nil, outage }

	handle := func(context.Context, domain.Message) error {
		t.Error("no message must be published when the claim fails")
		return nil
	}

	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, failingClaim, handle), testPolicy())

	count, err := dispatcher.DrainOnce(context.Background())
	if err == nil {
		t.Fatal("a failed claim must report an error")
	}
	if count != 0 {
		t.Errorf("processed messages = %d, want 0", count)
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventClaimFailed {
		t.Errorf("event = %q, want %q", got, domain.EventClaimFailed)
	}
}
