package tests

import (
	"context"
	"testing"
)

// TestRoleCaseNeverSilentlyDropsAGrant: `Accountant` and `accountant` are ONE
// role.
//
// # Why this test exists (ADR 013)
//
// `NewRole` normalises the role name and `NewIdentity` normalises the ones it
// receives — but assignment goes through a distinct path. If that path does not
// normalise, the role is kept under `Accountant`, looked up under `accountant`,
// and grants NOTHING.
//
// The fault is of the worst kind: it produces no error at all. An administrator
// sees the role assigned in the interface, the person concerned gets a 403, and
// both are right not to understand. Nothing in the logs mentions a case
// difference.
func TestRoleCaseNeverSilentlyDropsAGrant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	if err := mod.DefineRole(ctx, "Accountant", []string{"billing.invoice.cancel"}); err != nil {
		t.Fatalf("defining the role: %v", err)
	}
	if err := mod.AssignRoles(ctx, id, []string{"  ACCOUNTANT  "}); err != nil {
		t.Fatalf("assigning the role: %v", err)
	}

	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.cancel")); err != nil {
		t.Fatalf("a role's case must never make a permission be lost: %v", err)
	}
}

// TestAssigningAnUndefinedRoleGrantsNothingAndFailsNot separates provisioning
// order from the security decision.
//
// Assigning a role that does not exist yet grants NOTHING — `Grants` finds no
// permission — but remains allowed, so that one can provision in whatever order
// one likes. Refusing would be a sequencing constraint dressed up as a security
// rule, and it would fail a perfectly valid data import whose rows are not
// sorted.
//
// The property that matters is the second one: the assignment must grant
// nothing as long as the role is not defined.
func TestAssigningAnUndefinedRoleGrantsNothingAndFailsNot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	if err := mod.AssignRoles(ctx, id, []string{"never-defined"}); err != nil {
		t.Fatalf("the assignment must not depend on the provisioning order: %v", err)
	}
	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.cancel")); err == nil {
		t.Fatal("an undefined role must grant no permission")
	}

	// Once defined, it grants — without reassignment.
	if err := mod.DefineRole(ctx, "never-defined", []string{"billing.invoice.cancel"}); err != nil {
		t.Fatalf("late definition of the role: %v", err)
	}
	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.cancel")); err != nil {
		t.Fatalf("the role defined after the fact must grant: %v", err)
	}
}

// TestAssigningRolesToAnUnknownIdentityIsRefused forbids the silent success.
//
// A silent success would make the administrator believe the right is in place.
// It is the same fault as closing a non-existent account without saying so, and
// it is discovered at the same moment: when someone cannot do what they were
// nonetheless granted.
func TestAssigningRolesToAnUnknownIdentityIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	if err := mod.AssignRoles(ctx, "nobody", []string{"accountant"}); err == nil {
		t.Error("assigning a role to an unknown identity must be refused")
	}
	if err := mod.AssignRoles(ctx, "", []string{"accountant"}); err == nil {
		t.Error("assigning a role to an empty identity must be refused")
	}
}
