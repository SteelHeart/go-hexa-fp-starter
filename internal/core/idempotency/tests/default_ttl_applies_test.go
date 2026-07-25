package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestDefaultTTLApplies : sans option, le module retient une fenêtre de rejeu par
// défaut plutôt qu'aucune. Un TTL nul rendrait le module inopérant en silence.
func TestDefaultTTLApplies(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	clk.advance(23 * time.Hour)
	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("à 23 h, la réservation par défaut doit tenir, reçu %v", err)
	}
}
