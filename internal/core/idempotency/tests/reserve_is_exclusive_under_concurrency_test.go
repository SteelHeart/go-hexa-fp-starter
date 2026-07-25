package tests

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
)

// TestReserveIsExclusiveUnderConcurrency : le contrat de ports.Reserve exige que
// deux appels concurrents sur une clé libre ne l'obtiennent JAMAIS tous les deux.
// C'est la seule promesse du module, donc le seul test qui la prouve.
func TestReserveIsExclusiveUnderConcurrency(t *testing.T) {
	t.Parallel()

	mod := newMemoryModule(t, newClock(), "1h")
	req := request("k1", "charge")

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
		t.Errorf("réservations accordées = %d, attendu exactement 1", granted)
	}
	if refused != racers-1 {
		t.Errorf("refus ErrInFlight = %d, attendu %d", refused, racers-1)
	}
}
