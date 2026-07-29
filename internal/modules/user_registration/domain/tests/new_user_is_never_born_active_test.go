package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestNewUserIsNeverBornActive: an account is born PENDING.
//
// Being born active would be a fail-open: at that instant, the address is NOT
// proven. Anyone could register with somebody else's address and immediately
// have a working account in their name.
//
// This is "deny by default" applied to the life cycle, and it is the kind of
// decision that a redesign of the registration form undoes without noticing.
func TestNewUserIsNeverBornActive(t *testing.T) {
	t.Parallel()

	user := domain.NewUser(
		"user-42",
		validEmail(t, "alice@example.com"),
		domain.NewPasswordHash("$argon2id$..."),
		time.Now(),
	)

	if user.Status != domain.StatusPending {
		t.Errorf("state = %q, want %q", user.Status, domain.StatusPending)
	}
	if user.CanAuthenticate() {
		t.Error("an account whose address is not proven must NOT be able to sign in")
	}
}
