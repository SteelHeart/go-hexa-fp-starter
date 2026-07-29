//go:build integration

package integration

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	redisidem "github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/redis"
)

// TestIdempotencyRedisOnlyOneConcurrentReservationWins exercises the same
// exclusivity as the Postgres driver, against a radically different engine.
//
// Both drivers carry the same contract and obtain it by unrelated means:
// `ON CONFLICT … WHERE expires_at < now()` on one side, `SET NX` on the other.
// Verifying one says nothing about the other — which is precisely what a port
// as a function type makes possible, and what it makes mandatory to test
// twice.
//
// This test is the only one in the repository that touches Redis: the `cache`
// package and this driver had no test, at any level (#37).
func TestIdempotencyRedisOnlyOneConcurrentReservationWins(t *testing.T) {
	ctx := ctxTest(t)
	client := redisClient(t)

	namespace := unique(t, "integration-idem-redis")
	store := redisidem.New(client, namespace, time.Hour)

	key := domain.Key("shared-key")
	req := domain.Request{Key: key, Fingerprint: domain.Fingerprint(map[string]string{"a": "b"})}

	t.Cleanup(func() {
		client.Del(ctxTest(t), namespace+":"+key.String())
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
