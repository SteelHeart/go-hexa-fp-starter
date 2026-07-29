package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestUnknownSubjectAndWrongSecretAreIndistinguishable closes the account
// existence oracle.
//
// # What the fault produces when it is there
//
// A form that answers "this identifier does not exist" on one side and
// "incorrect password" on the other is an enumeration service: you submit a
// thousand addresses to it, note which ones answer differently, and you know
// exactly where to concentrate the effort. It is the most widespread mistake in
// this domain, and it is invisible on review because each message is, taken
// alone, more helpful.
//
// The three causes — malformed subject, unknown subject, wrong secret — must
// return the SAME error. The test compares the three against each other, not
// against an expected value: indistinguishability is the property, not the
// wording.
func TestUnknownSubjectAndWrongSecretAreIndistinguishable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	register(t, mod, subject)

	cases := map[string]struct{ subject, secret string }{
		"unknown subject":   {"nobody@example.com", secret},
		"wrong secret":      {subject, "another-long-secret"},
		"malformed subject": {"   ", secret},
		"empty secret":      {subject, ""},
	}

	messages := make(map[string]bool)
	for name, tc := range cases {
		_, err := mod.Authenticate(ctx, tc.subject, tc.secret)
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("%s: want ErrInvalidCredentials, got %v", name, err)
		}
		messages[err.Error()] = true
	}

	if len(messages) != 1 {
		t.Fatalf("the refusals must be indistinguishable; %d distinct messages: %v",
			len(messages), messages)
	}
}
