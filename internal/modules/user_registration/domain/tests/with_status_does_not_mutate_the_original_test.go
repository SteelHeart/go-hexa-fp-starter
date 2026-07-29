package tests

import (
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestWithStatusDoesNotMutateTheOriginal: `WithStatus` returns a COPY.
//
// Immutability is not a decorative principle here. A value shared between a
// cache decorator, an audit log and the business pipeline would be modified for
// everyone at once — and the audit would record a state the user never had at
// the time of the recorded fact.
func TestWithStatusDoesNotMutateTheOriginal(t *testing.T) {
	t.Parallel()

	original := domain.NewUser(
		"user-42",
		validEmail(t, "alice@example.com"),
		domain.NewPasswordHash("$argon2id$..."),
		time.Now(),
	)

	active := original.WithStatus(domain.StatusActive)

	if original.Status != domain.StatusPending {
		t.Errorf("the original was mutated: %q", original.Status)
	}
	if active.Status != domain.StatusActive {
		t.Errorf("the copy carries %q, want %q", active.Status, domain.StatusActive)
	}
	if active.ID != original.ID || active.Email.String() != original.Email.String() {
		t.Error("the copy must keep the other fields identical")
	}
}
