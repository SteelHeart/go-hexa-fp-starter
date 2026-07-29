package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestLockIsReleasedEvenWhenTheJobFails: work that fails still releases the
// election.
//
// Without that release, the first error of a task would freeze it until the
// replica died. The remedy would be worse than the disease: a task that fails
// once must retry on the next tick, not disappear.
func TestLockIsReleasedEvenWhenTheJobFails(t *testing.T) {
	t.Parallel()

	var released bool
	acquire := func(context.Context, domain.TaskName) (bool, error) { return true, nil }
	release := func(context.Context, domain.TaskName) error { released = true; return nil }

	log := &reportLog{}
	runner := newRunner(t, acquire, release, log)
	outage := errors.New("work failed")
	runner.RunOnce(context.Background(), application.Scheduled{
		Task: task("purge"),
		Job:  func(context.Context) error { return outage },
	})

	if !released {
		t.Error("the election must be released even after a failure of the work")
	}
	outcome, found := log.last()
	if !found {
		t.Fatal("no report")
	}
	if outcome.Event != domain.EventFailed {
		t.Errorf("event = %q, want %q", outcome.Event, domain.EventFailed)
	}
	if !errors.Is(outcome.Err, outage) {
		t.Errorf("cause = %v, want the original outage", outcome.Err)
	}
}
