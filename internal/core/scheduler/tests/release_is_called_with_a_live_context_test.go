package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestReleaseIsCalledWithALiveContext: the release must not inherit an already
// dead context.
//
// The trap is real and frequent: the work exhausts its timeout, the execution
// context is cancelled, and the release made with that same context fails
// instantly. The lock then stays taken until it expires — that is to say until
// the replica dies, for a session lock — and the task no longer runs at all. An
// overrun timeout would become a permanent outage.
func TestReleaseIsCalledWithALiveContext(t *testing.T) {
	t.Parallel()

	var releaseErr error
	acquire := func(context.Context, domain.TaskName) (bool, error) { return true, nil }
	release := func(ctx context.Context, _ domain.TaskName) error {
		releaseErr = ctx.Err()
		return nil
	}

	log := &reportLog{}
	runner := newRunner(t, acquire, release, log)

	// A task whose timeout is tiny and whose work waits for cancellation: the
	// execution context is therefore dead by the time of the release.
	expired := domain.Task{Name: "slow", Every: time.Hour, Timeout: time.Millisecond}
	runner.RunOnce(context.Background(), application.Scheduled{
		Task: expired,
		Job: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})

	if releaseErr != nil {
		t.Errorf("Release received a cancelled context (%v): the lock would stay taken", releaseErr)
	}
}
