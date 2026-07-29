package tests

import (
	"context"
	"testing"
)

// TestMemorizedResponseIsCopied: without a copy, the caller would keep a
// reference on the store's memory and could alter a memorised response. A replay
// would then return something other than the first call, silently.
func TestMemorizedResponseIsCopied(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "payload")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	response := []byte("original")
	if err := mod.Complete(ctx, req.Key, response); err != nil {
		t.Fatalf("memorisation: %v", err)
	}
	response[0] = 'X' // the caller reuses its buffer

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if string(replay.Response) != "original" {
		t.Errorf("memorised response = %q, altered by the caller", replay.Response)
	}

	replay.Response[0] = 'Y' // and the other way round
	second, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if string(second.Response) != "original" {
		t.Errorf("memorised response = %q, altered by a previous replay", second.Response)
	}
}
