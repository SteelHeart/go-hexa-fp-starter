package tests

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestReserveIsExclusiveUnderConcurrency: the ports.Reserve contract demands
// that two concurrent calls on a free key NEVER both obtain it. This is the
// module's only promise, hence the only test that proves it.
func TestReserveIsExclusiveUnderConcurrency(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	req := request("k1", "payload")

	const racers = 16
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		granted int
		refused int
	)
	wg.Add(racers)
	for range racers {
		go func() {
			defer wg.Done()
			_, err := mod.Reserve(context.Background(), req)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				granted++
			case errors.Is(err, domain.ErrInFlight):
				refused++
			}
		}()
	}
	wg.Wait()

	if granted != 1 {
		t.Errorf("granted reservations = %d, want exactly 1", granted)
	}
	if refused != racers-1 {
		t.Errorf("ErrInFlight refusals = %d, want %d", refused, racers-1)
	}
}
