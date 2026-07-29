package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestEnqueueThenClaimReturnsTheMessage(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()

	id, err := mod.Enqueue(ctx, domain.NewMessage{Type: "user.registered.v1", AggregateID: "u1"})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	claimed, err := mod.Claim(ctx, 10)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != id {
		t.Fatalf("Claim returned %d message(s), want 1 carrying %s", len(claimed), id)
	}
	if claimed[0].Status != domain.StatusPending {
		t.Errorf("initial status = %q, want pending", claimed[0].Status)
	}
}
