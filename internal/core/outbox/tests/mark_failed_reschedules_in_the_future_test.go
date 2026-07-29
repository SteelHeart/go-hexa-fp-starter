package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestMarkFailedReschedulesInTheFuture: after a failure, the message must NOT
// be immediately claimable again — otherwise the worker loops at full speed on
// a broken message.
func TestMarkFailedReschedulesInTheFuture(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()
	if _, err := mod.Enqueue(ctx, domain.NewMessage{Type: "t"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	claimed, _ := mod.Claim(ctx, 1)

	attempt := domain.NextAttempt(claimed[0],
		domain.RetryPolicy{MaxAttempts: 5, BaseBackoff: time.Second},
		fixedNow(), "network unavailable")
	if err := mod.MarkFailed(ctx, attempt); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	again, _ := mod.Claim(ctx, 10)
	if len(again) != 0 {
		t.Errorf("the message became claimable again immediately after a failure")
	}
	if count, _ := mod.PendingCount(ctx); count != 1 {
		t.Errorf("a message awaiting a retry must stay pending, count=%d", count)
	}
}
