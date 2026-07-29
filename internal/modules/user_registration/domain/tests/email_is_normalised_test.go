package tests

import "testing"

// TestEmailIsNormalised: the address is lower-cased and stripped of its
// surrounding whitespace.
//
// Without normalisation, `Alice@Example.com ` and `alice@example.com` would
// create TWO accounts. The user would not understand why their sign-in fails,
// and support would discover duplicates impossible to merge cleanly.
//
// It is also what makes the uniqueness index in the database genuinely
// effective: it bears on a single canonical form, not on a case variant.
func TestEmailIsNormalised(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"upper case":         "Alice@Example.COM",
		"surrounding spaces": "  alice@example.com  ",
		"both":               "\tALICE@Example.Com \n",
		"already normalised": "alice@example.com",
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := validEmail(t, raw).String(); got != "alice@example.com" {
				t.Errorf("NewEmail(%q) = %q, want the canonical form", raw, got)
			}
		})
	}
}
