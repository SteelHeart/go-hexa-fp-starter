package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestMarkFailedReschedulesInTheFuture : après un échec, le message ne doit PAS
// être immédiatement rejouable — sinon le worker boucle à plein régime sur un
// message cassé.
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
		fixedNow(), "réseau indisponible")
	if err := mod.MarkFailed(ctx, attempt); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	again, _ := mod.Claim(ctx, 10)
	if len(again) != 0 {
		t.Errorf("le message est redevenu réservable immédiatement après un échec")
	}
	if count, _ := mod.PendingCount(ctx); count != 1 {
		t.Errorf("un message en attente de réessai doit rester pending, count=%d", count)
	}
}
