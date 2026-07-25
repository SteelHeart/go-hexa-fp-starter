package tests

import (
	"context"
	"testing"
)

// TestRecordIsCalledOncePerFact : le pilote log n'écrit qu'une ligne par fait. Un
// doublon dans un journal d'audit fait douter de tout le journal.
func TestRecordIsCalledOncePerFact(t *testing.T) {
	t.Parallel()

	mod, buf := newLogModule(t)
	if err := mod.Record(context.Background(), completeEntry()); err != nil {
		t.Fatalf("Record: %v", err)
	}

	lines := 0
	for _, b := range buf.Bytes() {
		if b == '\n' {
			lines++
		}
	}
	if lines != 1 {
		t.Errorf("lignes écrites = %d, attendu 1", lines)
	}
}
