package tests

import (
	"context"
	"testing"
)

// TestReleaseNeverFreesACompletedKey: freeing a completed key would reopen the
// door to the replay the module exists to close.
func TestReleaseNeverFreesACompletedKey(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "payload")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte("ok")); err != nil {
		t.Fatalf("memorisation: %v", err)
	}
	if err := mod.Release(ctx, req.Key); err != nil {
		t.Fatalf("release: %v", err)
	}

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("reservation after the release: %v", err)
	}
	if !replay.Replayed {
		t.Error("the memorisation must survive a release")
	}
}
