package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestPermissionShapeIsEnforced: `domain.resource.action`, in lowercase,
// exactly three segments.
//
// # What the constraint avoids
//
// A permission is a string: nothing prevents writing `admin` on one side and
// `Admin` on the other. Both would coexist, one would grant, the other would
// not, and the defect would show up as a "they have the right but it refuses" —
// impossible to diagnose without comparing two strings by eye.
//
// The refusal happens at CONSTRUCTION: the field is private, so an invalid
// permission cannot exist outside the domain.
func TestPermissionShapeIsEnforced(t *testing.T) {
	t.Parallel()

	refused := []string{
		"",
		"admin",                      // one segment
		"billing.invoice",            // two
		"billing.invoice.cancel.now", // four
		"Billing.Invoice.Cancel",     // capitals — normalised, hence accepted
		"billing..cancel",            // empty segment
		"1billing.invoice.cancel",    // starts with a digit
		"billing.invoice.can-cel",    // hyphen
		"billing invoice cancel",     // spaces
	}
	for _, raw := range refused {
		_, err := domain.NewPermission(raw)
		if raw == "Billing.Invoice.Cancel" {
			if err != nil {
				t.Errorf("the case must be NORMALISED, not refused: %q → %v", raw, err)
			}
			continue
		}
		if !errors.Is(err, domain.ErrIncomplete) {
			t.Errorf("permission %q: want ErrIncomplete, got %v", raw, err)
		}
	}

	accepted, err := domain.NewPermission("  BILLING.Invoice.Cancel  ")
	if err != nil {
		t.Fatalf("valid permission refused: %v", err)
	}
	if accepted.String() != "billing.invoice.cancel" {
		t.Fatalf("want normalisation to `billing.invoice.cancel`, got %q", accepted.String())
	}
}

// TestDefiningARoleRefusesTheWholeSetOnOneBadPermission guards the atomicity of
// the definition.
//
// A half-defined role would be worse than a refused role: it would grant
// something, without anyone knowing what. The test also records that the faulty
// definition leaves NOTHING behind it.
func TestDefiningARoleRefusesTheWholeSetOnOneBadPermission(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	err := mod.DefineRole(ctx, "accountant", []string{
		"billing.invoice.cancel",
		"not a permission",
		"billing.invoice.read",
	})
	if !errors.Is(err, domain.ErrIncomplete) {
		t.Fatalf("want ErrIncomplete, got %v", err)
	}

	if err := mod.AssignRoles(ctx, id, []string{"accountant"}); err != nil {
		t.Fatalf("assignment: %v", err)
	}
	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.cancel")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("the definition failed: no permission must have been granted, got %v", err)
	}
}

// TestRoleGrantsWithoutWildcard: the comparison is exact, no wildcard.
//
// A `billing.*` would save a few lines of configuration and make it impossible
// to answer "who can cancel an invoice?" otherwise than by running the code.
// The question comes up in an audit, and it comes up about permissions one did
// not write oneself.
func TestRoleGrantsWithoutWildcard(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)
	grant(t, mod, id, "reader", "billing.invoice.read")

	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.read")); err != nil {
		t.Fatalf("the granted permission must hold: %v", err)
	}

	for _, other := range []string{"billing.invoice.cancel", "billing.report.read", "admin.user.read"} {
		if err := mod.Authorize(ctx, id, permission(t, other)); !errors.Is(err, domain.ErrForbidden) {
			t.Errorf("%q is not granted: want ErrForbidden, got %v", other, err)
		}
	}
}

// TestRoleAbsorbsDuplicatesAndSortsPermissions fixes the canonical form of a
// role.
//
// Duplicates are ABSORBED rather than refused: the same permission twice
// expresses the same intent, and refusing would fail a data import over a
// harmless redundancy.
//
// Sorting makes the equality of two roles observable and the messages
// comparable from one run to the next — a map's order changes on every walk.
func TestRoleAbsorbsDuplicatesAndSortsPermissions(t *testing.T) {
	t.Parallel()

	read := permission(t, "billing.invoice.read")
	cancel := permission(t, "billing.invoice.cancel")

	role, err := domain.NewRole("  Accountant  ", []domain.Permission{read, cancel, read})
	if err != nil {
		t.Fatalf("role: %v", err)
	}
	if role.Name != "accountant" {
		t.Fatalf("the role name must be normalised, got %q", role.Name)
	}
	if len(role.Permissions) != 2 {
		t.Fatalf("duplicates must be absorbed, got %v", role.Permissions)
	}
	if role.Permissions[0].String() != "billing.invoice.cancel" {
		t.Fatalf("the permissions must be sorted, got %v", role.Permissions)
	}

	if _, err := domain.NewRole("   ", nil); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("a role without a name: want ErrIncomplete, got %v", err)
	}
	if _, err := domain.NewRole("accountant", []domain.Permission{{}}); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("a permission that was never built: want ErrIncomplete, got %v", err)
	}
}
