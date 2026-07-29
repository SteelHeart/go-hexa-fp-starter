package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEmailIsMaskedBeforeLogging: an email address is personal data and is NEVER
// logged in clear (rules/securite.md §5).
//
// The masked form must stay useful for diagnosis — the domain part stays
// visible, which is enough to tell a corporate mail outage from a general
// problem — without allowing the address to be reconstructed.
//
// A log is kept for a long time, often outside the perimeter of the database,
// and read by humans during an incident. That is precisely the place where a
// leak goes unnoticed.
func TestEmailIsMaskedBeforeLogging(t *testing.T) {
	t.Parallel()

	masked := validEmail(t, "alice.martin@example.com").Masked()

	if strings.Contains(masked, "alice.martin") {
		t.Errorf("masked form = %q: the local part must be hidden", masked)
	}
	if !strings.Contains(masked, "example.com") {
		t.Errorf("masked form = %q: the domain must stay readable for diagnosis", masked)
	}
	if !strings.HasPrefix(masked, "a") {
		t.Errorf("masked form = %q: the first letter helps tell two accounts apart", masked)
	}

	// An address that was never constructed must not reveal anything either.
	var neverConstructed domain.Email
	if got := neverConstructed.Masked(); strings.Contains(got, "@") {
		t.Errorf("masked empty address = %q, want an opaque marker", got)
	}
}
