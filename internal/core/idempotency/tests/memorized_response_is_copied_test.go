package tests

import (
	"context"
	"testing"
)

// TestMemorizedResponseIsCopied : sans recopie, l'appelant garderait une référence
// sur la mémoire du magasin et pourrait altérer une réponse mémorisée. Un rejeu
// rendrait alors autre chose que le premier appel, silencieusement.
func TestMemorizedResponseIsCopied(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	response := []byte("original")
	if err := mod.Complete(ctx, req.Key, response); err != nil {
		t.Fatalf("mémorisation: %v", err)
	}
	response[0] = 'X' // l'appelant réutilise son tampon

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("rejeu: %v", err)
	}
	if string(replay.Response) != "original" {
		t.Errorf("réponse mémorisée = %q, altérée par l'appelant", replay.Response)
	}

	replay.Response[0] = 'Y' // et dans l'autre sens
	second, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("second rejeu: %v", err)
	}
	if string(second.Response) != "original" {
		t.Errorf("réponse mémorisée = %q, altérée par un rejeu précédent", second.Response)
	}
}
