package tests

import (
	"context"
	"testing"
)

// TestReleaseAllowsRetryAfterFailure : sans libération, une erreur transitoire
// rendrait l'opération impossible jusqu'à expiration de la clé — le remède serait
// pire que le mal.
func TestReleaseAllowsRetryAfterFailure(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if err := mod.Release(ctx, req.Key); err != nil {
		t.Fatalf("libération: %v", err)
	}

	again, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("après libération, la clé doit être réservable: %v", err)
	}
	if again.Replayed {
		t.Error("une clé libérée n'a rien mémorisé : Replayed doit être faux")
	}
}
