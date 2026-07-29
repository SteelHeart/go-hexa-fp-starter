package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestConcurrentRegistrationsNeverDuplicateAnAddress: two simultaneous
// registrations on the same address, only one wins.
//
// # Why this test exists
//
// The use case checks availability THEN writes. Between the two, there is a
// window: two concurrent requests both cross it, and both conclude that the
// address is free. This is the classic "check then act" defect, and it never
// shows up in development — it appears under load, in production, in the shape
// of two accounts for one and the same person.
//
// Only the STORE can settle it, because only it holds the lock. This test locks
// that guarantee down for the `memory` driver; the SQL driver will get it from
// its uniqueness constraint, and will have to pass the same test.
func TestConcurrentRegistrationsNeverDuplicateAnAddress(t *testing.T) {
	t.Parallel()

	const attempts = 16

	publisher := &spyPublisher{}
	mod := newModule(t, publisher)

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		conflicts int
	)

	start := make(chan struct{})
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // releases everyone at the same time

			_, failure, ok := mod.Register(context.Background(), domain.RegistrationCommand{
				Email:    "race@example.com",
				Password: validPassword,
			}).Get()

			mu.Lock()
			defer mu.Unlock()
			switch {
			case ok:
				succeeded++
			case failure.Code == domain.CodeEmailAlreadyExists:
				conflicts++
			default:
				t.Errorf("unexpected failure: %v", failure)
			}
		}()
	}
	close(start)
	wg.Wait()

	if succeeded != 1 {
		t.Errorf("successful registrations = %d, want exactly 1 — address uniqueness is not held", succeeded)
	}
	if conflicts != attempts-1 {
		t.Errorf("conflicts = %d, want %d — the losers must be refused explicitly", conflicts, attempts-1)
	}
}
