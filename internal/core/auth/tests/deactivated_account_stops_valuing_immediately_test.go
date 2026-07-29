package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestDeactivatedAccountStopsValuingImmediately covers ALL THREE doors at once.
//
// # Why all three, and not one
//
// `Active` is consulted in three places — authentication, resolving a token,
// and authorisation. Testing only one would let the other two drift, and the
// fault would be the worst of its kind: an account displayed as closed that
// keeps working through a path nobody looked at.
//
// The token is issued BEFORE the closure, on purpose: it is a token already in
// circulation, the one an attacker holds at the moment you react. No expiry
// comes into play.
func TestDeactivatedAccountStopsValuingImmediately(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	id := register(t, mod, subject)
	cancel := permission(t, "billing.invoice.cancel")
	grant(t, mod, id, "accountant", "billing.invoice.cancel")

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentication: %v", err)
	}

	if err := mod.Deactivate(ctx, id); err != nil {
		t.Fatalf("closing the account: %v", err)
	}

	if _, err := mod.Authenticate(ctx, subject, secret); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("closed account, authentication: want ErrInvalidCredentials, got %v", err)
	}
	if _, err := mod.Verify(ctx, session.Token); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Errorf("closed account, token already issued: want ErrTokenUnknown, got %v", err)
	}
	if err := mod.Authorize(ctx, id, cancel); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("closed account, authorisation: want ErrForbidden, got %v", err)
	}
}

// TestDeactivationIsIdempotentAndReversible guards both directions of the
// gesture.
//
// Idempotent: two administrators reacting to the same incident must not cancel
// each other out. Reversible: the closure is sometimes a mistake, and a module
// that only knew how to close would have that repaired by hand in the store —
// hence without a trace, and by someone who now has direct access to the
// accounts.
func TestDeactivationIsIdempotentAndReversible(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	for range 2 {
		if err := mod.Deactivate(ctx, id); err != nil {
			t.Fatalf("repeated closure: %v", err)
		}
	}

	if err := mod.Reactivate(ctx, id); err != nil {
		t.Fatalf("reopening: %v", err)
	}
	if _, err := mod.Authenticate(ctx, subject, secret); err != nil {
		t.Fatalf("the reopened account must authenticate: %v", err)
	}
}

// TestDeactivatingAnUnknownIdentityIsRefused forbids silently closing an
// imaginary account.
//
// A success on an unknown identifier would make the administrator believe they
// have just closed the compromised account — when they picked the wrong row,
// and the real account is still open.
func TestDeactivatingAnUnknownIdentityIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	if err := mod.Deactivate(ctx, "nobody"); err == nil {
		t.Error("closing an unknown identity must be refused")
	}
	if err := mod.Deactivate(ctx, ""); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("empty identity: want ErrIncomplete, got %v", err)
	}
}
