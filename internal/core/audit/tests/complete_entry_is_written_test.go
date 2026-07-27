package tests

import (
	"context"
	"testing"
)

// TestCompleteEntryIsWritten : le pilote log écrit les quatre champs qui
// répondent à « qui a fait quoi, quand, sur quoi ».
func TestCompleteEntryIsWritten(t *testing.T) {
	t.Parallel()

	mod, buf := newLogModule(t)
	if err := mod.Record(context.Background(), completeEntry()); err != nil {
		t.Fatalf("Record: %v", err)
	}

	line := decodeJournal(t, buf)
	expected := map[string]string{
		"actor":       "user-42",
		"action":      "user.registered",
		"entity_type": "user",
		"entity_id":   "42",
	}
	for field, want := range expected {
		if got, _ := line[field].(string); got != want {
			t.Errorf("%s = %v, attendu %q", field, line[field], want)
		}
	}
}
