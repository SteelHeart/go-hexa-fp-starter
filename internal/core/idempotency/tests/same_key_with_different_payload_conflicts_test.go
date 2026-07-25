package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestSameKeyWithDifferentPayloadConflicts : la même clé doit désigner la même
// requête. On refuse plutôt que de deviner laquelle des deux est la bonne.
func TestSameKeyWithDifferentPayloadConflicts(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()

	if _, err := mod.Reserve(ctx, request("k1", map[string]int{"montant": 100})); err != nil {
		t.Fatalf("première réservation refusée: %v", err)
	}
	_, err := mod.Reserve(ctx, request("k1", map[string]int{"montant": 999}))
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("erreur = %v, attendu ErrConflict", err)
	}
}
