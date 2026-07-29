package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestInprocDriverRefusesOverlap: the driver with no dependency elects nobody
// between replicas, but it does refuse local OVERLAP — and that is not nothing.
//
// A task slower than its period would otherwise pile executions up on every
// tick until the process was saturated. The outage would arrive long after
// going live, under load, and would look like a memory leak.
func TestInprocDriverRefusesOverlap(t *testing.T) {
	t.Parallel()

	mod := newInprocModule(t)
	ctx := context.Background()
	const name domain.TaskName = "purge"

	first, err := mod.Acquire(ctx, name)
	if err != nil {
		t.Fatalf("first election: %v", err)
	}
	if !first {
		t.Fatal("the first election must be granted")
	}

	second, err := mod.Acquire(ctx, name)
	if err != nil {
		t.Fatalf("second election: %v", err)
	}
	if second {
		t.Error("a task already in progress must not be relaunched")
	}

	if releaseErr := mod.Release(ctx, name); releaseErr != nil {
		t.Fatalf("release: %v", releaseErr)
	}
	again, err := mod.Acquire(ctx, name)
	if err != nil {
		t.Fatalf("election after release: %v", err)
	}
	if !again {
		t.Error("after release, the task must be re-electable")
	}
}
