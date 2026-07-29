package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestDispatcherStopsCleanlyOnCancellation: cancellation is the NORMAL end of a
// worker.
//
// A clean shutdown must not look like an outage, neither in the exit code nor
// in the alerts. Otherwise every redeployment produces one, and the team learns
// to ignore the dispatcher's errors — including the real ones, those that
// signal a definitively lost event.
//
// The test does not wait: it cancels as soon as the first publication is
// observed.
func TestDispatcherStopsCleanlyOnCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	published := make(chan struct{}, 1)
	claim := func(context.Context, int) ([]domain.Message, error) {
		return []domain.Message{pending("m-1", 0)}, nil
	}
	handle := func(context.Context, domain.Message) error {
		select {
		case published <- struct{}{}:
		default:
		}
		return nil
	}

	observed := &spy{}
	dispatcher := newDispatcher(t, dispatcherPorts(observed, claim, handle), testPolicy())

	stopped := make(chan error, 1)
	go func() { stopped <- dispatcher.Run(ctx) }()

	<-published // the loop really is running
	cancel()

	if err := <-stopped; err != nil {
		t.Errorf("Run = %v, a requested shutdown must return nil", err)
	}
}
