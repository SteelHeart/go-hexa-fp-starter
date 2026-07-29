// Package tests holds the BLACK BOX tests of the auth module: they only use the
// public API, exactly as an HTTP or CLI surface would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for the black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is checked, without having to open it.
//
// # No mocking library
//
// The ports are function types (ADR 003): a test double is a three-line
// closure. That is what makes a mocking library useless, hence forbidden
// (rules/dependances.md).
package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

const (
	// secret is a valid secret — at least twelve characters.
	secret = "a-secret-long-enough"

	// subject is the subject used by default.
	subject = "alice@example.com"

	// hashPrefix marks the dummy digest produced by the hashing double.
	//
	// The prefix is not decorative: it is what makes the difference between the
	// plain text and the digest OBSERVABLE, hence what makes the fact that the
	// plain text never reaches the store testable.
	hashPrefix = "digest:"
)

// clock is a clock the test advances by hand.
//
// The module receives its instant through a port — it never reads `time.Now()`.
// That is what makes it possible to check a session's expiry in one line,
// rather than by waiting twelve hours.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

// newClock freezes a reference instant.
func newClock() *clock {
	return &clock{now: time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC)}
}

// Now returns the test's current instant.
func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward.
func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// hashSecret is a DUMMY hash, reversible and instantaneous.
//
// Argon2id is deliberately slow: using it here would cost tens of milliseconds
// on every registration, to check nothing more. What these tests exercise is
// the secret's PATH — not the strength of the digest, which is exercised in
// internal/infrastructure/security.
func hashSecret(plain string) (string, error) { return hashPrefix + plain, nil }

// verifySecret compares a secret against the dummy digest.
func verifySecret(plain, encoded string) (bool, error) { return encoded == hashPrefix+plain, nil }

// deps assembles the module's dependencies around a clock.
func deps(c *clock) auth.Deps {
	return auth.Deps{HashSecret: hashSecret, VerifySecret: verifySecret, Now: c.Now}
}

// newModule builds the module on its default driver.
func newModule(t *testing.T, options map[string]any) (auth.Module, *clock) {
	t.Helper()

	c := newClock()
	mod, err := auth.New(
		config.Module{Enabled: true, Driver: "memory", Options: options},
		deps(c),
	)
	if err != nil {
		t.Fatalf("building the module: %v", err)
	}
	return mod, c
}

// register registers an identity and returns its identifier.
func register(t *testing.T, mod auth.Module, subj string) domain.IdentityID {
	t.Helper()

	identity, err := mod.Register(context.Background(), subj, secret)
	if err != nil {
		t.Fatalf("registering %q: %v", subj, err)
	}
	return identity.ID
}

// permission builds a permission or fails the test.
func permission(t *testing.T, raw string) domain.Permission {
	t.Helper()

	p, err := domain.NewPermission(raw)
	if err != nil {
		t.Fatalf("permission %q: %v", raw, err)
	}
	return p
}

// grant defines a role and assigns it to an identity.
func grant(t *testing.T, mod auth.Module, id domain.IdentityID, role string, permissions ...string) {
	t.Helper()

	ctx := context.Background()
	if err := mod.DefineRole(ctx, role, permissions); err != nil {
		t.Fatalf("defining role %q: %v", role, err)
	}
	if err := mod.AssignRoles(ctx, id, []string{role}); err != nil {
		t.Fatalf("assigning role %q: %v", role, err)
	}
}
