package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestCompleteAfterExpiryReportsNotReserved : l'opération métier a réussi, seule
// la mémorisation échoue. L'erreur doit le dire — l'appelant journalise, il
// n'annule pas.
func TestCompleteAfterExpiryReportsNotReserved(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	clk.advance(2 * time.Hour)

	err := mod.Complete(ctx, req.Key, []byte("ok"))
	if !errors.Is(err, domain.ErrNotReserved) {
		t.Errorf("Complete = %v, attendu ErrNotReserved", err)
	}
}
