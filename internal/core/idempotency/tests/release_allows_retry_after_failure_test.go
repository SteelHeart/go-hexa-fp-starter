package tests

import (
	"context"
	"testing"
)

// TestReleaseAllowsRetryAfterFailure: without a release, a transient error would
// make the operation impossible until the key expires — the remedy would be
// worse than the disease.
func TestReleaseAllowsRetryAfterFailure(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "payload")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	if err := mod.Release(ctx, req.Key); err != nil {
		t.Fatalf("release: %v", err)
	}

	again, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("after the release, the key must be reservable: %v", err)
	}
	if again.Replayed {
		t.Error("a released key memorised nothing: Replayed must be false")
	}
}
