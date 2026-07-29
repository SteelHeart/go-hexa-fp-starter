//go:build integration

package integration

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	pgidem "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/postgres"
)

// TestIdempotencyOnlyOneConcurrentReservationWins exercises exclusivity against
// a real engine.
//
// The `memory` driver has twenty-four tests proving this property — with a
// mutex, in a single process. The `postgres` driver entrusts it to
// `ON CONFLICT … DO UPDATE … WHERE expires_at < now()`, a construct whose
// correctness cannot be deduced: it can only be observed.
//
// What this guards: a mobile client re-emitting its request over an unstable
// network, and two replicas receiving it at the same time. If both obtain the
// reservation, the operation runs twice. On a payment, that is a double debit.
//
// Sixteen concurrent calls on the SAME key; exactly one must win.
func TestIdempotencyOnlyOneConcurrentReservationWins(t *testing.T) {
	ctx := ctxTest(t)
	p := pool(t)
	store := pgidem.New(p, time.Hour)

	key := domain.Key(unique(t, "integration-idem"))
	req := domain.Request{Key: key, Fingerprint: domain.Fingerprint(map[string]string{"a": "b"})}

	t.Cleanup(func() {
		_, _ = p.Exec(ctxTest(t), "DELETE FROM platform.idempotency_keys WHERE key = $1", key.String())
	})

	const calls = 16
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		winners  int
		inFlight int
		others   []error
		start    = make(chan struct{})
	)

	for range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// They all leave at the same instant: without this barrier, the
			// calls would run one after another and concurrency would never be
			// exercised.
			<-start

			res, err := store.Reserve(ctx, req)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && !res.Replayed:
				winners++
			case errors.Is(err, domain.ErrInFlight):
				inFlight++
			default:
				others = append(others, err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if len(others) > 0 {
		t.Fatalf("%d call(s) failed other than with \"already in flight\": %v", len(others), others)
	}
	if winners != 1 {
		t.Fatalf("%d reservations obtained out of %d concurrent calls, want exactly 1 — "+
			"the operation would run %d times", winners, calls, winners)
	}
	if inFlight != calls-1 {
		t.Errorf("%d call(s) refused as \"already in flight\", want %d", inFlight, calls-1)
	}
}
