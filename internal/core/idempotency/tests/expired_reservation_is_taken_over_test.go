package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestExpiredReservationIsTakenOver : une réservation abandonnée — processus tué
// entre Reserve et Complete — doit se reprendre. Sans reprise, un plantage
// bloquerait la clé jusqu'à sa purge et le client ne pourrait plus rien faire.
func TestExpiredReservationIsTakenOver(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Fatalf("avant expiration, attendu ErrInFlight, reçu %v", err)
	}

	clk.advance(time.Hour + time.Second)

	taken, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("après expiration, la clé doit être reprise: %v", err)
	}
	if taken.Replayed {
		t.Error("une réservation expirée n'a rien mémorisé")
	}
}
