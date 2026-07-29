package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestSubjectCaseNeverForksAnAccount: `Alice@Example.COM ` and
// `alice@example.com` are ONE identity, not two.
//
// # The fault, when the normalisation is missing
//
// Two accounts coexist. The registration succeeds twice, so nothing reports
// anything. Then someone signs in with the case of "the other" account, lands
// on an empty account, and nobody understands — least of all support, who can
// see the address in the database perfectly well.
//
// The normalisation is carried by the TYPE: `Subject`'s field is private, so a
// non-normalised subject cannot exist outside the domain.
func TestSubjectCaseNeverForksAnAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	register(t, mod, "  Alice@Example.COM ")

	if _, err := mod.Register(ctx, "alice@example.com", secret); !errors.Is(err, domain.ErrSubjectTaken) {
		t.Fatalf("the same address up to case: want ErrSubjectTaken, got %v", err)
	}

	for _, variant := range []string{"alice@example.com", "ALICE@EXAMPLE.COM", " Alice@Example.Com "} {
		if _, err := mod.Authenticate(ctx, variant, secret); err != nil {
			t.Errorf("authenticating with %q: %v", variant, err)
		}
	}
}

// TestMalformedSubjectIsRefusedBeforeTheStore guards the refusal UPSTREAM of
// the driver.
//
// The refusal happens upstream for two reasons: it costs no query, and it does
// not let an empty string reach a driver, where it would become a legitimate
// key — hence an account anyone could claim by sending nothing.
func TestMalformedSubjectIsRefusedBeforeTheStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	for _, raw := range []string{"", "   ", "\t", "alice bob@example.com"} {
		if _, err := mod.Register(ctx, raw, secret); !errors.Is(err, domain.ErrIncomplete) {
			t.Errorf("subject %q: want ErrIncomplete, got %v", raw, err)
		}
	}
}

// TestShortSecretIsRefused: twelve characters, and no composition rule.
//
// Length is the only constraint that really increases entropy. Composition
// rules — an uppercase letter, a digit, a special character — mostly push
// people to write `Password1!`, which satisfies all four and resists nothing.
func TestShortSecretIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	if _, err := mod.Register(ctx, subject, "eleven-char"); !errors.Is(err, domain.ErrIncomplete) {
		t.Fatalf("secret too short: want ErrIncomplete, got %v", err)
	}

	// Twelve characters with no uppercase, no digit, no special character:
	// accepted. The rule is the length, and it alone.
	if _, err := mod.Register(ctx, subject, "abcdefghijkl"); err != nil {
		t.Fatalf("no composition rule must apply: %v", err)
	}
}

// TestSubjectIsMaskedForLogs: a subject is personal data.
//
// It is never logged in clear (rules/securite.md §5) — and the authentication
// log is the one most readily exported to a third-party collector.
func TestSubjectIsMaskedForLogs(t *testing.T) {
	t.Parallel()

	subj, err := domain.NewSubject("alice@example.com")
	if err != nil {
		t.Fatalf("subject: %v", err)
	}

	masked := subj.Masked()
	if masked == subj.String() {
		t.Fatal("the masked form must not be the subject in clear")
	}
	if masked != "a***@example.com" {
		t.Fatalf("unexpected masked form: %q", masked)
	}

	short, err := domain.NewSubject("ab")
	if err != nil {
		t.Fatalf("short subject: %v", err)
	}
	if short.Masked() != "***" {
		t.Fatalf("a subject that is too short must disappear entirely, got %q", short.Masked())
	}
}
