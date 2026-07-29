package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestNonElectedReplicaDoesNotExecute is the most important test of the module.
//
// All the value of the scheduler is there: N replicas, one single execution.
// Without this refusal, a reminder goes out N times and an invoice is issued N
// times — defects the customer discovers before we do.
//
// And "not elected" is NOT an error: it is the nominal case on N-1 replicas, on
// every tick. The report must say so, otherwise the logs would contain nothing
// but that.
func TestNonElectedReplicaDoesNotExecute(t *testing.T) {
	t.Parallel()

	var executed bool
	acquire := func(context.Context, domain.TaskName) (bool, error) { return false, nil }
	release := func(context.Context, domain.TaskName) error {
		t.Error("Release must not be called when the election failed")
		return nil
	}

	log := &reportLog{}
	runner := newRunner(t, acquire, release, log)
	runner.RunOnce(context.Background(), application.Scheduled{
		Task: task("daily-reminder"),
		Job:  func(context.Context) error { executed = true; return nil },
	})

	if executed {
		t.Error("a non-elected replica must NOT run the work")
	}
	outcome, found := log.last()
	if !found {
		t.Fatal("no report")
	}
	if outcome.Event != domain.EventSkipped {
		t.Errorf("event = %q, want %q", outcome.Event, domain.EventSkipped)
	}
	if outcome.Err != nil {
		t.Errorf("\"not elected\" is not an error, got %v", outcome.Err)
	}
}
