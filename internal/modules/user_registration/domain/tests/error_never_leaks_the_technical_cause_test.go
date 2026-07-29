package tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestErrorNeverLeaksTheTechnicalCause: the technical detail is loggable, never
// returnable.
//
// An SQL error sent back to the client is a structure leak: names of tables, of
// columns, of constraints. It is a map of the schema handed over, and one of the
// first things an attacker looks for.
//
// The cause is therefore carried in an UNEXPORTED field, reachable through
// `Cause()` for the log, and absent from `Error()` which surfaces display.
func TestErrorNeverLeaksTheTechnicalCause(t *testing.T) {
	t.Parallel()

	const technicalSecret = "pq: duplicate key value violates unique constraint \"users_email_key\""
	cause := errors.New(technicalSecret)

	err := domain.NewError(domain.CodeEmailAlreadyExists, "cette adresse est déjà enregistrée").
		WithField("email").
		WithCause(cause)

	if strings.Contains(err.Error(), technicalSecret) {
		t.Errorf("Error() leaks the technical detail: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("Error() = %q: the faulty field must appear in it", err.Error())
	}
	if !errors.Is(err.Cause(), cause) {
		t.Error("Cause() must return the technical detail, for the log")
	}
}
