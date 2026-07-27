package tests

import (
	"context"
	"testing"
)

// TestReleaseNeverFreesACompletedKey : libérer une clé achevée rouvrirait la porte
// au rejeu que le module existe pour fermer.
func TestReleaseNeverFreesACompletedKey(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte("ok")); err != nil {
		t.Fatalf("mémorisation: %v", err)
	}
	if err := mod.Release(ctx, req.Key); err != nil {
		t.Fatalf("libération: %v", err)
	}

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("réservation après libération: %v", err)
	}
	if !replay.Replayed {
		t.Error("la mémorisation doit survivre à une libération")
	}
}
