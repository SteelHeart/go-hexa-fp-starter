package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestSecondReservationIsRefusedWhileInFlight: this is THE guarantee of the
// module. Without this refusal, two concurrent requests would both execute the
// operation.
func TestSecondReservationIsRefusedWhileInFlight(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("payment-1", map[string]int{"amount": 4200})

	first, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("first reservation refused: %v", err)
	}
	if first.Replayed {
		t.Fatal("the first reservation cannot be a replay")
	}

	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("second reservation = %v, want ErrInFlight", err)
	}
}
