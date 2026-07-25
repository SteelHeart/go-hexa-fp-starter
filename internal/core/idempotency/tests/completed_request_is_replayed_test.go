package tests

import (
	"context"
	"testing"
)

// TestCompletedRequestIsReplayed : après mémorisation, le rejeu rend la réponse
// du premier appel sans rien réexécuter.
func TestCompletedRequestIsReplayed(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte(`{"id":"user-42"}`)); err != nil {
		t.Fatalf("mémorisation: %v", err)
	}

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("le rejeu ne doit pas échouer: %v", err)
	}
	if !replay.Replayed {
		t.Fatal("Replayed doit être vrai : l'appelant ne doit PAS réexécuter")
	}
	if string(replay.Response) != `{"id":"user-42"}` {
		t.Errorf("réponse mémorisée = %q", replay.Response)
	}
}
