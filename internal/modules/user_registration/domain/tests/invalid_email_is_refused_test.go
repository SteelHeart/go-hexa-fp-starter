package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestInvalidEmailIsRefused: the Email type CANNOT carry an invalid value.
//
// The field is unexported and `NewEmail` is the only construction path: any
// function that receives an Email therefore no longer has to validate it. The
// type guarantees it, not a review convention — and this test is what makes that
// guarantee true.
//
// The "Name <a@b.c>" case deserves a mention: `mail.ParseAddress` happily
// accepts it. Without the explicit refusal, two different inputs would yield the
// same user, and the displayed name would come from unvalidated input.
func TestInvalidEmailIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"empty":                    "",
		"whitespace only":          "   ",
		"no at sign":               "alice.example.com",
		"no domain":                "alice@",
		"no local part":            "@example.com",
		"domain without a dot":     "alice@localhost",
		"form with a display name": "Alice <alice@example.com>",
		"two at signs":             "alice@@example.com",
		"inner space":              "ali ce@example.com",
		"too long":                 strings.Repeat("a", 250) + "@example.com",
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := codeOf(t, domain.NewEmail(raw)); got != domain.CodeInvalidEmail {
				t.Errorf("code = %q, want %q", got, domain.CodeInvalidEmail)
			}
		})
	}
}
