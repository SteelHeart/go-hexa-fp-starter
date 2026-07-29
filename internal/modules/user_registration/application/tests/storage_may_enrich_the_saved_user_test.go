package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestStorageMayEnrichTheSavedUser: it is the user RETURNED by the storage that
// carries on through the pipeline, not the one it was given.
//
// The difference matters as soon as the database has the last word on a value:
// an identifier produced by a sequence, a timestamp set by a `DEFAULT now()`, a
// state adjusted by a trigger. Ignoring the return value would publish an event
// carrying values the database never wrote.
func TestStorageMayEnrichTheSavedUser(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	deps := nominalDeps(observed)
	deps.SaveUser = func(_ context.Context, u domain.User) result.Result[domain.User, domain.Error] {
		observed.note("SaveUser")
		enriched := u.WithStatus(domain.StatusActive) // the database decided otherwise
		observed.saved = enriched
		return result.Ok[domain.User, domain.Error](enriched)
	}

	user := userOf(t, register(deps))

	if user.Status != domain.StatusActive {
		t.Errorf("state = %q: the return value of the storage was ignored", user.Status)
	}
}
