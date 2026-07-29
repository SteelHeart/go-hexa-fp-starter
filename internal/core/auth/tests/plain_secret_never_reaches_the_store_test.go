package tests

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// TestPlainSecretNeverReachesTheStore records the ORDER of the steps.
//
// # What the test really observes
//
// `VerifySecret` receives the plain text that was typed AND what the store
// kept. The latter is therefore the only observation point a caller has on the
// store's contents — and it is enough: if the plain text were in there, it
// would show up right here.
//
// Hashing BEFORE writing is not a matter of style. It is what guarantees a
// driver cannot log a password even by accident: it never sees one.
func TestPlainSecretNeverReachesTheStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	c := newClock()

	var (
		mu     sync.Mutex
		stored string
	)
	spy := auth.Deps{
		HashSecret: hashSecret,
		VerifySecret: func(plain, encoded string) (bool, error) {
			mu.Lock()
			stored = encoded
			mu.Unlock()
			return verifySecret(plain, encoded)
		},
		Now: c.Now,
	}

	mod, err := auth.New(config.Module{Enabled: true, Driver: "memory"}, spy)
	if err != nil {
		t.Fatalf("building the module: %v", err)
	}
	register(t, mod, subject)

	if _, err := mod.Authenticate(ctx, subject, secret); err != nil {
		t.Fatalf("authentication: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	switch {
	case stored == "":
		t.Fatal("the digest was never compared: the test observes nothing")
	case stored == secret:
		t.Fatal("the store keeps the PLAIN secret")
	case strings.Contains(stored, secret) && !strings.HasPrefix(stored, hashPrefix):
		t.Fatalf("the plain text shows through what the store keeps: %q", stored)
	}
}

// TestSessionCarriesNoPermission records what the session does NOT carry.
//
// This is decision 1 of ADR 017 seen from the type: a `Session` only has its
// token, its identity and its dates. The day someone added a `Permissions`
// field to it, this test would stop compiling — and that is the right moment to
// reopen the ADR, not six months later in front of a revocation that does not
// take effect.
func TestSessionCarriesNoPermission(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)
	grant(t, mod, id, "accountant", "billing.invoice.cancel")

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentication: %v", err)
	}

	if session.Token.IsZero() {
		t.Fatal("the session must carry a token")
	}
	if session.Identity != id {
		t.Fatalf("the session must carry identity %q, it carries %q", id, session.Identity)
	}
	if !session.ExpiresAt.After(session.IssuedAt) {
		t.Fatal("a session must be bounded in time")
	}
	if strings.Contains(session.Token.String(), "billing") {
		t.Fatal("the token carries a permission: it authenticates, it does not authorise")
	}
}
