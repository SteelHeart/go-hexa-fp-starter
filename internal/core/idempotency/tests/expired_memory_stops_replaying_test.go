package tests

import (
	"context"
	"testing"
	"time"
)

// TestExpiredMemoryStopsReplaying : la fenêtre de rejeu est finie, et c'est un
// arbitrage assumé. Au-delà du TTL, le rejeu recrée la ressource.
func TestExpiredMemoryStopsReplaying(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "30m")
	ctx := context.Background()
	req := request("k1", "charge")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("réservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte("ok")); err != nil {
		t.Fatalf("mémorisation: %v", err)
	}

	clk.advance(31 * time.Minute)

	after, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("réservation après expiration: %v", err)
	}
	if after.Replayed {
		t.Error("passé le TTL, la réponse ne doit plus être rejouée")
	}
}
