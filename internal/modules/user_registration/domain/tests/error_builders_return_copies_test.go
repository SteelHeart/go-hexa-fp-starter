package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestErrorBuildersReturnCopies: `WithField` and `WithCause` return a new error.
//
// An Error is a VALUE, not an interface, precisely so that the set of errors
// stays enumerable. If the builders mutated their receiver, a shared sentinel
// error — declared once at package level — would end up enriched with the field
// and the cause of the latest call. Two concurrent requests would then swap
// their technical details.
func TestErrorBuildersReturnCopies(t *testing.T) {
	t.Parallel()

	base := domain.NewError(domain.CodeInternal, "erreur interne")

	withField := base.WithField("email")
	withCause := base.WithCause(errors.New("detail"))

	if base.Field != "" {
		t.Errorf("WithField mutated the original: field = %q", base.Field)
	}
	if base.Cause() != nil {
		t.Error("WithCause mutated the original")
	}
	if withField.Field != "email" {
		t.Errorf("the copy carries %q, want \"email\"", withField.Field)
	}
	if withCause.Cause() == nil {
		t.Error("the copy must carry the cause")
	}
	if withField.Cause() != nil {
		t.Error("both copies must be independent of one another")
	}
}
