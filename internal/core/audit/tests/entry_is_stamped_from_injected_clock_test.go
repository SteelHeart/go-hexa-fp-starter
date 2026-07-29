package tests

import (
	"context"
	"testing"
	"time"
)

// TestEntryIsStampedFromInjectedClock: the instant comes from a port, never
// from time.Now(). Without that injection, no audit test would be deterministic
// and no retention would be verifiable.
func TestEntryIsStampedFromInjectedClock(t *testing.T) {
	t.Parallel()

	mod, buf := newLogModule(t)
	if err := mod.Record(context.Background(), completeEntry()); err != nil {
		t.Fatalf("Record: %v", err)
	}

	raw, _ := decodeLogLine(t, buf)["occurred_at"].(string)
	stamped, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t.Fatalf("unreadable timestamp (%q): %v", raw, err)
	}
	if !stamped.Equal(recordedAt()) {
		t.Errorf("occurred_at = %v, want %v", stamped, recordedAt())
	}
}
