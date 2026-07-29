package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestPublishedMessageIsMarkedDone: the nominal path, and the only one where
// the asynchronous chain is genuinely closed.
//
// Publishing without marking would republish the message on the next round,
// indefinitely: the consumer would receive the same event every two seconds,
// forever.
func TestPublishedMessageIsMarkedDone(t *testing.T) {
	t.Parallel()

	observed := &spy{}
	var delivered []domain.MessageID
	handle := func(_ context.Context, msg domain.Message) error {
		delivered = append(delivered, msg.ID)
		return nil
	}

	dispatcher := newDispatcher(t,
		dispatcherPorts(observed, claimOnce(pending("m-1", 0)), handle), testPolicy())

	count, err := dispatcher.DrainOnce(context.Background())
	if err != nil {
		t.Fatalf("DrainOnce: %v", err)
	}
	if count != 1 {
		t.Errorf("processed messages = %d, want 1", count)
	}
	if len(delivered) != 1 || delivered[0] != "m-1" {
		t.Errorf("published messages = %v, want [m-1]", delivered)
	}
	if len(observed.done) != 1 || observed.done[0] != "m-1" {
		t.Errorf("marked messages = %v, want [m-1]", observed.done)
	}
	if got := observed.lastOutcome(t).Event; got != domain.EventPublished {
		t.Errorf("event = %q, want %q", got, domain.EventPublished)
	}
}
