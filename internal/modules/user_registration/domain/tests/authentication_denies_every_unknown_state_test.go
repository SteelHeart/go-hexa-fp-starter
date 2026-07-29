package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestAuthenticationDeniesEveryUnknownState: only `active` allows signing in.
//
// The `default` of the switch refuses. This is not paranoia: the day someone
// adds a state — `suspended`, `deleted` — without completing the function, the
// default behaviour must be REFUSAL. The opposite would open a suspended account
// because of an oversight.
//
// The `exhaustive` linter fails the CI in that case; this test covers the
// situation where someone would disable the linter, or build a state by hand.
func TestAuthenticationDeniesEveryUnknownState(t *testing.T) {
	t.Parallel()

	base := domain.NewUser(
		"user-42",
		validEmail(t, "alice@example.com"),
		domain.NewPasswordHash("$argon2id$..."),
		time.Now(),
	)

	cases := map[domain.Status]bool{
		domain.StatusActive:  true,
		domain.StatusPending: false,
		domain.StatusBlocked: false,
		"invented state":     false,
		"":                   false,
	}

	for status, want := range cases {
		if got := base.WithStatus(status).CanAuthenticate(); got != want {
			t.Errorf("CanAuthenticate with state %q = %v, want %v", status, got, want)
		}
	}
}
