package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestSecondReservationIsRefusedWhileInFlight : c'est LA garantie du module. Sans
// ce refus, deux requêtes concurrentes exécuteraient toutes les deux l'opération.
func TestSecondReservationIsRefusedWhileInFlight(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("paiement-1", map[string]int{"montant": 4200})

	first, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("première réservation refusée: %v", err)
	}
	if first.Replayed {
		t.Fatal("la première réservation ne peut pas être un rejeu")
	}

	if _, err := mod.Reserve(ctx, req); !errors.Is(err, domain.ErrInFlight) {
		t.Errorf("seconde réservation = %v, attendu ErrInFlight", err)
	}
}
