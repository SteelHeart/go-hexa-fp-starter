package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestElectionFailureNeverExecutes: when in doubt, do not run.
//
// An unreachable database does not say "nobody holds the task", it says
// nothing. Running anyway would amount to assuming one is alone — a false
// assumption as soon as there are two replicas, and one that produces exactly
// the double execution the module exists to prevent.
//
// The fallback therefore goes in the safe direction: not running is benign,
// running twice is not.
func TestElectionFailureNeverExecutes(t *testing.T) {
	t.Parallel()

	outage := errors.New("database unreachable")
	acquire := func(context.Context, domain.TaskName) (bool, error) { return false, outage }
	release := func(context.Context, domain.TaskName) error {
		t.Error("Release must not be called when the election is down")
		return nil
	}

	var executed bool
	log := &reportLog{}
	runner := newRunner(t, acquire, release, log)
	runner.RunOnce(context.Background(), application.Scheduled{
		Task: task("billing"),
		Job:  func(context.Context) error { executed = true; return nil },
	})

	if executed {
		t.Error("an election that is down must NOT lead to an execution")
	}
	outcome, found := log.last()
	if !found {
		t.Fatal("an election outage must be reported, never swallowed")
	}
	if outcome.Event != domain.EventElectionFailed {
		t.Errorf("event = %q, want %q", outcome.Event, domain.EventElectionFailed)
	}
	if !errors.Is(outcome.Err, outage) {
		t.Errorf("cause = %v, want the original outage", outcome.Err)
	}
}
