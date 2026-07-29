package tests

import (
	"context"
	"testing"
)

// TestDistinctTasksDoNotBlockEachOther: two different tasks run in parallel.
// The exclusion bears on ONE task, never on the whole scheduler.
//
// A global lock would be simpler to write and catastrophic to operate: a long
// task would stop all the others from running, and the symptom — "some tasks no
// longer start" — would not point at the culprit.
func TestDistinctTasksDoNotBlockEachOther(t *testing.T) {
	t.Parallel()

	mod := newInprocModule(t)
	ctx := context.Background()

	elected, err := mod.Acquire(ctx, "purge")
	if err != nil || !elected {
		t.Fatalf("election of purge: elected=%v err=%v", elected, err)
	}

	elected, err = mod.Acquire(ctx, "reminders")
	if err != nil {
		t.Fatalf("election of reminders: %v", err)
	}
	if !elected {
		t.Error("another task must not be blocked by the first")
	}
}
