package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestExpiredSessionStopsVerifying checks expiry without waiting.
//
// # Why the use case re-checks what the store already knows
//
// The in-memory driver does NOT purge: an expired session stays there until
// restart. If `Verify` merely found it, it would go on being worth something
// indefinitely. Relying on the driver to return only valid sessions would make
// security depend on an implementation detail — and the first driver that
// purged differently would reopen the hole without anything reporting it.
//
// The bound is STRICT: a session expires AT its date, not after.
func TestExpiredSessionStopsVerifying(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, c := newModule(t, map[string]any{"session_ttl": "1h"})
	register(t, mod, subject)

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentication: %v", err)
	}

	c.Advance(59 * time.Minute)
	if _, err := mod.Verify(ctx, session.Token); err != nil {
		t.Fatalf("the session has not expired yet: %v", err)
	}

	c.Advance(time.Minute)
	if _, err := mod.Verify(ctx, session.Token); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Fatalf("expired session: want ErrTokenUnknown, got %v", err)
	}
}

// TestSessionTTLOptionIsHonoured records that the option is READ.
//
// A silently ignored configuration option is the defect that cost issue #93:
// the server started, mounted the driver, and said nothing. Here, a one-second
// lifetime must produce a session that does not survive one second — which the
// twelve-hour default would not do.
func TestSessionTTLOptionIsHonoured(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, c := newModule(t, map[string]any{"session_ttl": "1s"})
	register(t, mod, subject)

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentication: %v", err)
	}

	c.Advance(time.Second)
	if _, err := mod.Verify(ctx, session.Token); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Fatalf("the session_ttl option is ignored: %v", err)
	}
}
