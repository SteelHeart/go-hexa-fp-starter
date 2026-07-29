package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestRevokedPermissionIsRefusedOnTheNextCall is THE witness of ADR 017.
//
// # What it records
//
// A withdrawn permission stops granting on the NEXT call, without any token
// expiring and without anyone signing in again. The token stays valid
// throughout — the test checks that explicitly between the two authorisations,
// otherwise a refusal could come from an invalidated session and the
// demonstration would be empty.
//
// # Why this test exists (ADR 013)
//
// It would fail the day someone moved the permissions into the token, or added
// an unbounded cache in front of `Grants`. Both are tempting optimisations, and
// both reopen the same window: a revoked access that still works. And the day
// you revoke is the day you are in a hurry.
func TestRevokedPermissionIsRefusedOnTheNextCall(t *testing.T) {
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
	if err := mod.Authorize(ctx, id, cancel); err != nil {
		t.Fatalf("the permission has just been granted, it must hold: %v", err)
	}

	// Revocation: the role is REDEFINED without the permission. No token is
	// touched, no session is deleted.
	if err := mod.DefineRole(ctx, "accountant", nil); err != nil {
		t.Fatalf("revoking the permission: %v", err)
	}

	// The token is STILL worth something: that is what makes the refusal that
	// follows conclusive.
	if _, err := mod.Verify(ctx, session.Token); err != nil {
		t.Fatalf("the token should not have been affected by the revocation: %v", err)
	}

	if err := mod.Authorize(ctx, id, cancel); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("revoked permission: want ErrForbidden, got %v", err)
	}
}

// TestRevokedRoleIsRefusedOnTheNextCall records the same thing from the other
// end.
//
// Withdrawing the ROLE rather than the permission must produce the same
// immediate refusal. Both paths exist — you withdraw a right from everybody, or
// you withdraw somebody from a group — and covering only one of the two would
// let the other drift.
func TestRevokedRoleIsRefusedOnTheNextCall(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	id := register(t, mod, subject)
	cancel := permission(t, "billing.invoice.cancel")
	grant(t, mod, id, "accountant", "billing.invoice.cancel")

	if err := mod.Authorize(ctx, id, cancel); err != nil {
		t.Fatalf("the permission has just been granted, it must hold: %v", err)
	}

	if err := mod.AssignRoles(ctx, id, nil); err != nil {
		t.Fatalf("withdrawing the roles: %v", err)
	}

	if err := mod.Authorize(ctx, id, cancel); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("withdrawn role: want ErrForbidden, got %v", err)
	}
}
