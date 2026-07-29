package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestDefaultTTLApplies: without an option, the module keeps a default replay
// window rather than none. A zero TTL would make the module silently
// inoperative.
func TestDefaultTTLApplies(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "")
	ctx := context.Background()
	req := request("k1", "payload")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	clk.advance(23 * time.Hour)
	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("at 23 h, the default reservation must hold, got %v", err)
	}
}
