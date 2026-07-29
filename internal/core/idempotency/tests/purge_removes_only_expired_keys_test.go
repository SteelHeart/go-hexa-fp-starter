package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestPurgeRemovesOnlyExpiredKeys: the memory driver expires nothing on its own.
// Purge is therefore the only bound on the growth of the map, and it must never
// carry away a live reservation.
func TestPurgeRemovesOnlyExpiredKeys(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()

	for _, key := range []string{"old-1", "old-2"} {
		if _, err := mod.Reserve(ctx, request(key, "payload")); err != nil {
			t.Fatalf("reservation of %s: %v", key, err)
		}
	}
	clk.advance(2 * time.Hour)
	if _, err := mod.Reserve(ctx, request("recent", "payload")); err != nil {
		t.Fatalf("recent reservation: %v", err)
	}

	removed, err := mod.Purge(ctx)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if removed != 2 {
		t.Errorf("purged keys = %d, want 2", removed)
	}
	if _, err := mod.Reserve(ctx, request("recent", "payload")); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("the recent reservation should have survived the purge, got %v", err)
	}
}
