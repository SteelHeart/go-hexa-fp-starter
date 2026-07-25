package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestPurgeRemovesOnlyExpiredKeys : le pilote mémoire n'expire rien tout seul.
// Purge est donc la seule borne à la croissance de la carte, et elle ne doit
// jamais emporter une réservation vivante.
func TestPurgeRemovesOnlyExpiredKeys(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "1h")
	ctx := context.Background()

	for _, key := range []string{"vieille-1", "vieille-2"} {
		if _, err := mod.Reserve(ctx, request(key, "charge")); err != nil {
			t.Fatalf("réservation de %s: %v", key, err)
		}
	}
	clk.advance(2 * time.Hour)
	if _, err := mod.Reserve(ctx, request("recente", "charge")); err != nil {
		t.Fatalf("réservation récente: %v", err)
	}

	removed, err := mod.Purge(ctx)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if removed != 2 {
		t.Errorf("clés purgées = %d, attendu 2", removed)
	}
	if _, err := mod.Reserve(ctx, request("recente", "charge")); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("la réservation récente devait survivre à la purge, reçu %v", err)
	}
}
