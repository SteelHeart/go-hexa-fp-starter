package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestRegistrationReportsTheEmailFirst: when the address AND the password are
// both wrong, it is the address that is reported.
//
// The order is not arbitrary: it is the field the user fills in first, therefore
// the one they will correct first. Reporting the password first would have them
// fix one field only to discover afterwards that the previous one was at fault —
// two round trips instead of one.
//
// This is an ergonomics decision, taken once, and this test keeps it from
// getting lost at the first refactoring of the validation pipeline.
func TestRegistrationReportsTheEmailFirst(t *testing.T) {
	t.Parallel()

	bothWrong := domain.RegistrationCommand{Email: "not an address", Password: "short"}
	if got := codeOf(t, domain.ParseRegistration(bothWrong)); got != domain.CodeInvalidEmail {
		t.Errorf("code = %q, want %q first", got, domain.CodeInvalidEmail)
	}

	validAddress := domain.RegistrationCommand{Email: "alice@example.com", Password: "short"}
	if got := codeOf(t, domain.ParseRegistration(validAddress)); got != domain.CodeWeakPassword {
		t.Errorf("code = %q, want %q once the address is valid", got, domain.CodeWeakPassword)
	}
}
