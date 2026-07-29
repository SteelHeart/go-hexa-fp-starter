package tests

import (
	"context"
	"testing"
)

// TestCompletedRequestIsReplayed: after memorisation, the replay returns the
// response of the first call without re-executing anything.
func TestCompletedRequestIsReplayed(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	ctx := context.Background()
	req := request("k1", "payload")

	if _, err := mod.Reserve(ctx, req); err != nil {
		t.Fatalf("reservation: %v", err)
	}
	if err := mod.Complete(ctx, req.Key, []byte(`{"id":"user-42"}`)); err != nil {
		t.Fatalf("memorisation: %v", err)
	}

	replay, err := mod.Reserve(ctx, req)
	if err != nil {
		t.Fatalf("the replay must not fail: %v", err)
	}
	if !replay.Replayed {
		t.Fatal("Replayed must be true: the caller must NOT re-execute")
	}
	if string(replay.Response) != `{"id":"user-42"}` {
		t.Errorf("memorised response = %q", replay.Response)
	}
}
