package tests

import (
	"context"
	"testing"
	"time"
)

// TestExpiredMemoryStopsReplaying: the replay window is finite, and that is a
// deliberate trade-off. Beyond the TTL, the replay recreates the resource.
func TestExpiredMemoryStopsReplaying(t *testing.T) {
	t.Parallel()

	clk := newClock()
	mod := newMemoryModule(t, clk, "30m")
	ctx := context.Background()
	req := request("k1", "payload")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte("ok")); err != nil {
		t.Fatalf("memorisation: %v", err)
	}

	clk.advance(31 * time.Minute)

	after, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("reservation after expiry: %v", err)
	}
	if after.Replayed {
		t.Error("past the TTL, the response must no longer be replayed")
	}
}
