package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestPublishFailureFailsTheWholeRegistration is the other half of the outbox
// pattern, and the most counter-intuitive one.
//
// The user was written successfully. Yet, if recording the event fails, the
// WHOLE registration fails.
//
// The temptation would be to return a success while logging the incident: after
// all, the account exists. But the write and the event are in the SAME
// transaction: the failure triggers the rollback, so the account already no
// longer exists. Returning a success would lie to the caller about an account
// that has just vanished.
//
// A user created without their welcome event is a silently inconsistent state —
// and silence is the worst kind of defect.
func TestPublishFailureFailsTheWholeRegistration(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	deps := nominalDeps(observed)
	deps.PublishEvent = func(
		context.Context, string, string, any,
	) result.Result[domain.Ack, domain.Error] {
		observed.note("PublishEvent")
		return failing[domain.Ack](domain.CodeUnavailable, "outbox injoignable")
	}

	if got := codeOf(t, register(deps)); got != domain.CodeUnavailable {
		t.Errorf("code = %q, want %q", got, domain.CodeUnavailable)
	}
	if !observed.called("SaveUser") {
		t.Error("the write must indeed have been attempted before the publication")
	}
}
