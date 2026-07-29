package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestAuthorizeRefusesAnIncompleteDemand: the zero value opens nothing.
//
// # The case a permissive store would make catastrophic
//
// A `Permission{}` that was never built carries the empty string, and so does
// an empty `IdentityID`. Without this upstream refusal, both would reach the
// store where they would become legitimate keys — and the first row recorded
// under an empty key would grant access to anyone who sends nothing.
//
// The refusal is distinct from `ErrForbidden`: the request itself is malformed,
// which an HTTP surface translates into a 422 and not a 403.
func TestAuthorizeRefusesAnIncompleteDemand(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)
	grant(t, mod, id, "accountant", "billing.invoice.cancel")

	cases := map[string]struct {
		identity   domain.IdentityID
		permission domain.Permission
	}{
		"empty identity":     {"", permission(t, "billing.invoice.cancel")},
		"permission unbuilt": {id, domain.Permission{}},
		"both":               {"", domain.Permission{}},
	}

	for name, tc := range cases {
		err := mod.Authorize(ctx, tc.identity, tc.permission)
		if !errors.Is(err, domain.ErrIncomplete) {
			t.Errorf("%s: want ErrIncomplete, got %v", name, err)
		}
		if errors.Is(err, domain.ErrForbidden) {
			t.Errorf("%s: a malformed request is not a permission refusal", name)
		}
	}
}

// TestAuthorizeRefusesAnUnknownIdentity: an invented identifier grants nothing.
//
// Deny by default all the way: `Grants` returns `false` for what it does not
// know, and the use case translates that into `ErrForbidden`. The opposite
// mistake — treating "unknown" as "no known restriction" — is how an access
// control opens up entirely in one line.
func TestAuthorizeRefusesAnUnknownIdentity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	err := mod.Authorize(ctx, "invented-identifier", permission(t, "billing.invoice.cancel"))
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("unknown identity: want ErrForbidden, got %v", err)
	}
}

// TestVerifyAndAuthorizeStayTwoCalls records the SEPARATION of the two gestures.
//
// `Verify` returns an identity and says nothing about permissions; `Authorize`
// returns a refusal or nothing and says nothing about the identity. Merging
// them into a `verifyAndAuthorize(token, permission)` would bring permissions
// back within the token's scope through the back door, and the next natural
// optimisation would be to put them in there for good.
//
// The test records that a PERFECTLY valid identity obtains nothing without a
// granted permission.
func TestVerifyAndAuthorizeStayTwoCalls(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentication: %v", err)
	}

	identity, err := mod.Verify(ctx, session.Token)
	if err != nil {
		t.Fatalf("the token is valid: %v", err)
	}
	if identity.ID != id {
		t.Fatalf("want identity %q, got %q", id, identity.ID)
	}

	if err := mod.Authorize(ctx, identity.ID, permission(t, "billing.invoice.cancel")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("authenticated is not authorised: want ErrForbidden, got %v", err)
	}
}
