package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestCompleteAfterExpiryReportsNotReserved: the business operation succeeded,
// only the memorisation fails. The error must say so — the caller logs, it does
// not roll back.
func TestCompleteAfterExpiryReportsNotReserved(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()
	req := request("k1", "payload")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	clk.advance(2 * time.Hour)

	err := mod.Complete(ctx, req.Key, []byte("ok"))
	if !errors.Is(err, domain.ErrNotReserved) {
		t.Errorf("Complete = %v, want ErrNotReserved", err)
	}
}
