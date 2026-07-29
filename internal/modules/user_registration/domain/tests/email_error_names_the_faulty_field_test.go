package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEmailErrorNamesTheFaultyField: a validation error names its field.
//
// Without `Field`, an HTTP surface cannot place the message under the right form
// control: it displays a generic banner and the user has to work out for
// themselves what is wrong. The field is what turns an error into a possible
// correction.
func TestEmailErrorNamesTheFaultyField(t *testing.T) {
	t.Parallel()

	err := failureOf(t, domain.NewEmail("not an address"))
	if err.Field != "email" {
		t.Errorf("field = %q, want \"email\"", err.Field)
	}
	if err.Message == "" {
		t.Error("the message intended for the user must not be empty")
	}
}
