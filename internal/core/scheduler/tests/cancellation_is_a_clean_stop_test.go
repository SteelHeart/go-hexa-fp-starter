package tests

import (
	"context"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestCancellationIsACleanStop: cancellation is the NORMAL end of a scheduler.
//
// A clean stop must not look like an outage, neither in the return code, nor in
// the reports. Otherwise every redeployment fills the alerts, and the team
// learns to ignore the errors of the scheduler — including the real ones.
//
// The test does not sleep: it finishes as soon as the first execution happens,
// triggered by a very short period, then cancels.
func TestCancellationIsACleanStop(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	executed := make(chan struct{}, 1)
	acquire, release := alwaysElected()
	log := &reportLog{}
	runner := newRunner(t, acquire, release, log)

	quick := domain.Task{Name: "quick", Every: time.Millisecond, Timeout: time.Second}
	stopped := make(chan error, 1)
	go func() {
		stopped <- runner.Run(ctx, []application.Scheduled{{
			Task: quick,
			Job: func(context.Context) error {
				select {
				case executed <- struct{}{}:
				default:
				}
				return nil
			},
		}})
	}()

	<-executed // the loop really is running
	cancel()

	if err := <-stopped; err != nil {
		t.Errorf("Run = %v, a requested stop must return nil", err)
	}
	if log.count(domain.EventSucceeded) == 0 {
		t.Error("no successful execution was reported")
	}
	if log.count(domain.EventFailed) != 0 {
		t.Errorf("a clean stop must produce no failure, got %d", log.count(domain.EventFailed))
	}
}
