package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/subscriber"
)

// TestAReplayedEnvelopeNeverRepeatsTheEffect is the WITNESS of issue #9.
//
// # What it observes
//
// The same envelope, delivered ten times, produces the effect ONLY ONCE — and
// the nine other deliveries acknowledge without an error, because a recognised
// replay is not a breakdown.
//
// # Why this test is indispensable
//
// Every transport here is "at least once". The same envelope therefore arrives
// twice as soon as an acknowledgement is lost — which is commonplace, not
// exceptional. Without this guard, a welcome email leaves twice: the visible
// symptom. On the day a consumer debits a card, it will be the debit, and it
// will not be visible before the statement.
func TestAReplayedEnvelopeNeverRepeatsTheEffect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	effect := &counter{}
	handler := subscriber.Once(newGuard(t), effect.handler())
	env := envelope("evt-1")

	for attempt := 1; attempt <= 10; attempt++ {
		if err := handler(ctx, env); err != nil {
			t.Fatalf("delivery no. %d: a recognised replay is not a breakdown — %v", attempt, err)
		}
	}

	if effect.total() != 1 {
		t.Fatalf("the effect took place %d times, want exactly 1", effect.total())
	}
}

// TestTwoDistinctEnvelopesBothProduceTheirEffect guards the other half.
//
// A guard that blocked EVERYTHING would be green on the previous test and
// perfectly useless. It is the most common shape of a broken guard: the one
// that refuses everything looks in every respect like the one that only lets
// through what it should.
func TestTwoDistinctEnvelopesBothProduceTheirEffect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	effect := &counter{}
	handler := subscriber.Once(newGuard(t), effect.handler())

	for _, id := range []string{"evt-1", "evt-2", "evt-3"} {
		if err := handler(ctx, envelope(id)); err != nil {
			t.Fatalf("envelope %s: %v", id, err)
		}
	}

	if effect.total() != 3 {
		t.Fatalf("three distinct envelopes produced %d effects, want 3", effect.total())
	}
}

// TestConcurrentDeliveriesProduceASingleEffect: the SIMULTANEOUS replay.
//
// The sequential case would pass even with a naive "already seen?" check placed
// before the execution. It is the simultaneous delivery that makes the window
// appear — and behind two replicas, it is the norm, not the exception.
//
// The losing deliveries return an ERROR, and that is intended: another replica
// is handling the envelope and may still fail. Acknowledging them would lose
// the event if the other one gives up.
func TestConcurrentDeliveriesProduceASingleEffect(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	effect := &counter{}
	handler := subscriber.Once(newGuard(t), effect.handler())
	env := envelope("evt-simultaneous")

	const deliveries = 16
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
	)

	wg.Add(deliveries)
	for range deliveries {
		go func() {
			defer wg.Done()
			if err := handler(ctx, env); err == nil {
				mu.Lock()
				succeeded++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if effect.total() != 1 {
		t.Fatalf("%d simultaneous deliveries produced %d effects, want 1", deliveries, effect.total())
	}
	if succeeded < 1 {
		t.Fatal("no delivery succeeded: the event would be lost")
	}
}
