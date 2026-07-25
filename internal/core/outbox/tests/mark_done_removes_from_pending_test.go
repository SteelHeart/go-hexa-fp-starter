package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

func TestMarkDoneRemovesFromPending(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()
	id, _ := mod.Enqueue(ctx, domain.NewMessage{Type: "t"})

	if count, _ := mod.PendingCount(ctx); count != 1 {
		t.Fatalf("PendingCount = %d, attendu 1", count)
	}
	if err := mod.MarkDone(ctx, id); err != nil {
		t.Fatalf("MarkDone: %v", err)
	}
	if count, _ := mod.PendingCount(ctx); count != 0 {
		t.Errorf("PendingCount après MarkDone = %d, attendu 0", count)
	}
}
