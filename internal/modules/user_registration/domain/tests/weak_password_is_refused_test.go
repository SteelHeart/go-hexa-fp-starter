package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestWeakPasswordIsRefused: the password bounds are security decisions, each
// for a different reason.
//
//   - **High minimum, with no composition rule.** A long passphrase resists
//     better than a short "P@ssw0rd!", and composition requirements produce
//     predictable passwords — the user puts an upper case letter at the start
//     and a "!" at the end.
//   - **Maximum.** Argon2id on a ten megabyte input is a denial of service
//     handed over for free: the hashing cost is paid by the SERVER, not by the
//     attacker.
//   - **Repetitiveness.** "aaaaaaaaaaaa" is twelve characters long and worth
//     nothing.
func TestWeakPasswordIsRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"empty":                     "",
		"too short":                 "short123",
		"eleven characters":         "elevenchar1",
		"whitespace only":           strings.Repeat(" ", 20),
		"too repetitive":            "aaaaaaaaaaaaaa",
		"three distinct characters": "abcabcabcabcabc",
		"too long":                  strings.Repeat("passphrase-word-", 20),
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := codeOf(t, domain.NewRawPassword(raw)); got != domain.CodeWeakPassword {
				t.Errorf("code = %q, want %q", got, domain.CodeWeakPassword)
			}
		})
	}
}
