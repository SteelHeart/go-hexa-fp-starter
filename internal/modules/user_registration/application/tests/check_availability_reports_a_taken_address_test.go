package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestCheckAvailabilityReportsATakenAddress: a taken address is not available,
// and a breakdown is not an availability.
//
// The second point is the real trap. If an unreachable database returned
// "available", a breakdown would turn the form into a duplicate-making machine —
// duplicates the uniqueness constraint would then refuse, with a message
// incomprehensible to the user who has just seen a green tick.
//
// The error therefore travels up as it is: "I do not know" is not "yes".
func TestCheckAvailabilityReportsATakenAddress(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	taken := application.NewCheckEmailAvailability(
		func(context.Context, domain.Email) result.Result[bool, domain.Error] {
			return result.Ok[bool, domain.Error](true)
		},
	)
	available, _, ok := taken(ctx, validAddress).Get()
	if !ok {
		t.Fatal("a taken address must return a success carrying false, not an error")
	}
	if available {
		t.Error("an already registered address must NOT be announced as available")
	}

	brokenDown := application.NewCheckEmailAvailability(
		func(context.Context, domain.Email) result.Result[bool, domain.Error] {
			return failing[bool](domain.CodeUnavailable, "base injoignable")
		},
	)
	_, err, ok := brokenDown(ctx, validAddress).Get()
	if ok {
		t.Fatal("a breakdown must not turn into an availability")
	}
	if err.Code != domain.CodeUnavailable {
		t.Errorf("code = %q, want %q", err.Code, domain.CodeUnavailable)
	}
}
