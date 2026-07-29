package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestStorageFailurePublishesNothing: a failed write publishes no event at all.
//
// This is half of the outbox pattern. Publishing "user registered" while the
// write has failed would produce a GHOST EVENT: consumers would send a welcome
// email, create a profile, bill — for a user who does not exist.
//
// A ghost event is more serious than a lost one: the lost one is replayed, the
// ghost one is cleaned up by hand, across several systems.
func TestStorageFailurePublishesNothing(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	deps := nominalDeps(observed)
	deps.SaveUser = func(context.Context, domain.User) result.Result[domain.User, domain.Error] {
		observed.note("SaveUser")
		return failing[domain.User](domain.CodeUnavailable, "base injoignable")
	}

	if got := codeOf(t, register(deps)); got != domain.CodeUnavailable {
		t.Errorf("code = %q, want %q", got, domain.CodeUnavailable)
	}
	if observed.called("PublishEvent") {
		t.Error("no event must be published when the write has failed")
	}
}
