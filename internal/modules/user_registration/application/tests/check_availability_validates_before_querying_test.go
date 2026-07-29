package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestCheckAvailabilityValidatesBeforeQuerying: the address is validated BEFORE
// the storage is queried.
//
// This use case is the entry point of a form field: it is called on every
// keystroke, without authentication, from any client. Without prior validation,
// every character typed would become a query — a free way of making the database
// work.
//
// It returns `true` for AVAILABLE: the double negative "not taken" is a classic
// source of mistakes, and inverting it would make every address free.
func TestCheckAvailabilityValidatesBeforeQuerying(t *testing.T) {
	t.Parallel()

	queried := false
	check := application.NewCheckEmailAvailability(
		func(context.Context, domain.Email) result.Result[bool, domain.Error] {
			queried = true
			return result.Ok[bool, domain.Error](false)
		},
	)
	ctx := context.Background()

	_, err, ok := check(ctx, "not an address").Get()
	if ok {
		t.Fatal("an invalid address must be refused")
	}
	if err.Code != domain.CodeInvalidEmail {
		t.Errorf("code = %q, want %q", err.Code, domain.CodeInvalidEmail)
	}
	if queried {
		t.Error("the storage must NOT be queried for an invalid address")
	}

	available, _, ok := check(ctx, validAddress).Get()
	if !ok {
		t.Fatal("a valid and free address must return a success")
	}
	if !available {
		t.Error("an address that is NOT taken must be returned as AVAILABLE")
	}
}
