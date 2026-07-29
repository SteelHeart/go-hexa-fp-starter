package tests

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestSameSubjectRegistersOnlyOnceUnderConcurrency closes the window between
// the check and the write.
//
// # The window this test aims at
//
// The use case does NOT check uniqueness before writing, deliberately: between
// a check and a write there is an interval that two simultaneous requests both
// cross. It is the store, which holds the lock, that decides — exactly as an
// SQL uniqueness constraint would.
//
// A sequential test would pass even with the fault. Sixteen concurrent
// registrations make it appear.
func TestSameSubjectRegistersOnlyOnceUnderConcurrency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	const attempts = 16
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		ids       = make(map[domain.IdentityID]bool)
	)

	wg.Add(attempts)
	for range attempts {
		go func() {
			defer wg.Done()
			identity, err := mod.Register(ctx, subject, secret)

			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				succeeded++
				ids[identity.ID] = true
				return
			}
			if !errors.Is(err, domain.ErrSubjectTaken) {
				t.Errorf("want refusal ErrSubjectTaken, got %v", err)
			}
		}()
	}
	wg.Wait()

	if succeeded != 1 {
		t.Fatalf("%d registrations succeeded on the same subject, want exactly 1", succeeded)
	}
	if len(ids) != 1 {
		t.Fatalf("%d distinct identifiers created, want 1", len(ids))
	}
}

// TestEachAccountGetsItsOwnIdentifier demands two accounts, two identifiers.
//
// A reused identifier would make someone carry another person's permissions.
// The test also checks that the identity is born ACTIVE — unlike
// `user_registration`, whose account is born `pending`. The nuance is real:
// `auth` only creates an identity on a request already authorised by its
// caller, whereas a public registration must be confirmed.
func TestEachAccountGetsItsOwnIdentifier(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	seen := make(map[domain.IdentityID]bool)
	for _, subj := range []string{"alice@example.com", "bob@example.com", "carol@example.com"} {
		identity, err := mod.Register(ctx, subj, secret)
		if err != nil {
			t.Fatalf("registering %q: %v", subj, err)
		}
		if seen[identity.ID] {
			t.Fatalf("identifier reused for %q: %q", subj, identity.ID)
		}
		if !identity.Active {
			t.Fatalf("an authentication identity is born active; %q is not", subj)
		}
		if len(identity.Roles) != 0 {
			t.Fatalf("an identity is born WITHOUT a role; %q carries %v", subj, identity.Roles)
		}
		seen[identity.ID] = true
	}
}
