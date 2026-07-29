package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
)

// TestCatchUpDrainsMoreThanOneBatch: a FULL batch means « there are more ».
//
// Waiting for the next tick would advance the catch-up by a single batch per
// period. If the input rate exceeds one batch per round — which happens as soon
// as a consumer comes back after an outage — the backlog would never be
// absorbed and would grow instead.
//
// The catch-up is bounded: without a bound, a hundred thousand backlogged
// messages would monopolise the loop, the `select` would no longer be reached,
// and the worker would ignore the shutdown request until it had drained
// everything.
func TestCatchUpDrainsMoreThanOneBatch(t *testing.T) {
	t.Parallel()

	const batchSize = 2

	var mu sync.Mutex
	rounds := 0
	// Two full batches, then a partial batch that signals the end of the backlog.
	claim := func(context.Context, int) ([]domain.Message, error) {
		mu.Lock()
		defer mu.Unlock()
		rounds++
		switch rounds {
		case 1:
			return []domain.Message{pending("m-1", 0), pending("m-2", 0)}, nil
		case 2:
			return []domain.Message{pending("m-3", 0), pending("m-4", 0)}, nil
		default:
			return []domain.Message{pending("m-5", 0)}, nil
		}
	}

	observed := &spy{}
	handle := func(context.Context, domain.Message) error { return nil }

	policy := testPolicy()
	policy.BatchSize = batchSize
	dispatcher := newDispatcher(t, dispatcherPorts(observed, claim, handle), policy)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Three successive calls reproduce what a polling round does: chain as long
	// as the batch comes back full.
	for range 3 {
		count, err := dispatcher.DrainOnce(ctx)
		if err != nil {
			t.Fatalf("DrainOnce: %v", err)
		}
		if count == 0 {
			break
		}
	}

	if len(observed.done) != 5 {
		t.Errorf("processed messages = %d, want 5 — the backlog was not caught up", len(observed.done))
	}
}
