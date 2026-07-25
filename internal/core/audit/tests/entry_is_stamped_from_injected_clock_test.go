package tests

import (
	"context"
	"testing"
	"time"
)

// TestEntryIsStampedFromInjectedClock : l'instant vient d'un port, jamais de
// time.Now(). Sans cette injection, aucun test d'audit ne serait déterministe et
// aucune rétention ne serait vérifiable.
func TestEntryIsStampedFromInjectedClock(t *testing.T) {
	t.Parallel()

	mod, buf := newLogModule(t)
	if err := mod.Record(context.Background(), completeEntry()); err != nil {
		t.Fatalf("Record: %v", err)
	}

	raw, _ := decodeJournal(t, buf)["occurred_at"].(string)
	stamped, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t.Fatalf("horodatage illisible (%q): %v", raw, err)
	}
	if !stamped.Equal(recordedAt()) {
		t.Errorf("occurred_at = %v, attendu %v", stamped, recordedAt())
	}
}
