package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestValidRegistrationNeedsNoFurtherValidation: a valid command produces a type
// that CARRIES its validity.
//
// This is the boundary crossing, and it happens only once. Downstream, no step
// revalidates: it could not, since it no longer receives a `string` but domain
// types. That is what eliminates repeated defensive validation — the kind added
// "just in case" that ends up diverging from one place to another.
func TestValidRegistrationNeedsNoFurtherValidation(t *testing.T) {
	t.Parallel()

	cmd := domain.RegistrationCommand{
		Email:    "  Alice@Example.COM ",
		Password: "correct horse battery staple",
	}

	valid, err, ok := domain.ParseRegistration(cmd).Get()
	if !ok {
		t.Fatalf("valid command refused: %v", err)
	}
	if got := valid.Email.String(); got != "alice@example.com" {
		t.Errorf("address = %q, want the normalised form", got)
	}
	if valid.Password.Expose() != cmd.Password {
		t.Error("the password must cross the validation without being altered")
	}
}
