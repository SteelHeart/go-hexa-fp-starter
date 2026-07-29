package tests

import (
	"testing"
)

// TestWithTimeDoesNotMutateTheEntry: domain.Entry is a value. A method that
// mutated its receiver would make two records of the same fact diverge.
func TestWithTimeDoesNotMutateTheEntry(t *testing.T) {
	t.Parallel()

	original := completeEntry()
	_ = original.WithTime(recordedAt())

	if !original.At.IsZero() {
		t.Errorf("WithTime mutated the original entry: At = %v", original.At)
	}
}
