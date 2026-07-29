package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestRevokingASessionIsImmediateAndIdempotent guards both properties of a
// sign-out.
//
// Immediate: that is what one expects from a sign-out, and it is what a
// self-contained signed token cannot do without a revocation list — hence
// without falling back on the store it claimed to avoid.
//
// Idempotent: a client who signs out twice has done nothing wrong. Failing the
// second call would produce an error nobody would know how to handle, and that
// everyone would end up ignoring — including when it reports something else.
func TestRevokingASessionIsImmediateAndIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	register(t, mod, subject)

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentication: %v", err)
	}
	if _, err := mod.Verify(ctx, session.Token); err != nil {
		t.Fatalf("the token has just been issued: %v", err)
	}

	if err := mod.Revoke(ctx, session.Token); err != nil {
		t.Fatalf("revocation: %v", err)
	}
	if _, err := mod.Verify(ctx, session.Token); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Fatalf("revoked token: want ErrTokenUnknown, got %v", err)
	}

	if err := mod.Revoke(ctx, session.Token); err != nil {
		t.Fatalf("revoking twice must not be an error: %v", err)
	}
}

// TestRevokingOneSessionSparesTheOthers prevents a sign-out from becoming a
// global one.
//
// Two sign-ins of the same account — a phone and a workstation — produce two
// tokens. Signing out of one must not sign out the other. The opposite
// mistake, indexing the session on the identity rather than on the token, is
// only noticed the day someone signs in from two devices.
func TestRevokingOneSessionSparesTheOthers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	register(t, mod, subject)

	first, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("first authentication: %v", err)
	}
	second, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("second authentication: %v", err)
	}
	if first.Token.Equals(second.Token) {
		t.Fatal("two authentications must produce two distinct tokens")
	}

	if err := mod.Revoke(ctx, first.Token); err != nil {
		t.Fatalf("revocation: %v", err)
	}
	if _, err := mod.Verify(ctx, second.Token); err != nil {
		t.Fatalf("the second session should not have been touched: %v", err)
	}
}

// TestVerifyRefusesAnEmptyToken: the zero value opens nothing.
//
// A token that was never built is an empty string. Without this refusal, it
// would become a legitimate key in the store — and the first session recorded
// under that key would open the door to anyone who sends no token at all.
func TestVerifyRefusesAnEmptyToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	if _, err := mod.Verify(ctx, domain.Token{}); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Fatalf("empty token: want ErrTokenUnknown, got %v", err)
	}
	if err := mod.Revoke(ctx, domain.Token{}); !errors.Is(err, domain.ErrIncomplete) {
		t.Fatalf("revoking an empty token: want ErrIncomplete, got %v", err)
	}
}
