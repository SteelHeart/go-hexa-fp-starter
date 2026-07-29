package tests

import (
	"context"
	"testing"
)

// TestRecordIsCalledOncePerFact: the log driver writes only one line per fact.
// A duplicate in an audit log casts doubt on the whole log.
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
		t.Errorf("lines written = %d, want 1", lines)
	}
}
