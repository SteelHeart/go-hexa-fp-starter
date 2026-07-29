package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestExpiredReservationIsTakenOver: an abandoned reservation — process killed
// between Reserve and Complete — must be taken over. Without take-over, a crash
// would block the key until it is purged and the client could do nothing more.
func TestExpiredReservationIsTakenOver(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()
	req := request("k1", "payload")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Fatalf("before expiry, want ErrInFlight, got %v", err)
	}

	clk.advance(time.Hour + time.Second)

	taken, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("after expiry, the key must be taken over: %v", err)
	}
	if taken.Replayed {
		t.Error("an expired reservation memorised nothing")
	}
}
