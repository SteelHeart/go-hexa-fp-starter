package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestClaimIsExclusive vérifie le contrat le plus important du port : deux
// réservations concurrentes ne retournent JAMAIS le même message. Le pilote
// postgres l'obtient par FOR UPDATE SKIP LOCKED ; le pilote memory doit le
// garantir aussi, sinon il mentirait sur le comportement de production.
func TestClaimIsExclusive(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t)
	ctx := context.Background()
	if _, err := mod.Enqueue(ctx, domain.NewMessage{Type: "t"}); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	first, err := mod.Claim(ctx, 10)
	if err != nil || len(first) != 1 {
		t.Fatalf("première réservation: %d message(s), err=%v", len(first), err)
	}
	second, err := mod.Claim(ctx, 10)
	if err != nil {
		t.Fatalf("seconde réservation: %v", err)
	}
	if len(second) != 0 {
		t.Errorf("un message déjà réservé a été rendu une seconde fois: %d", len(second))
	}
}
