package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestTakenEmailIsRefusedBeforeHashing: an already taken address is refused
// BEFORE hashing.
//
// The order is not cosmetic. Argon2id is deliberately slow and memory hungry —
// that is its reason to exist. Hashing before checking availability would hand
// anyone a way of saturating the server by submitting, over and over, an address
// they know already exists.
//
// The test therefore verifies both things: the right error code, AND the fact
// that the costly port was not called.
func TestTakenEmailIsRefusedBeforeHashing(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	deps := nominalDeps(observed)
	deps.EmailIsTaken = func(context.Context, domain.Email) result.Result[bool, domain.Error] {
		observed.note("EmailIsTaken")
		return result.Ok[bool, domain.Error](true)
	}

	if got := codeOf(t, register(deps)); got != domain.CodeEmailAlreadyExists {
		t.Errorf("code = %q, want %q", got, domain.CodeEmailAlreadyExists)
	}
	if observed.called("HashPassword") {
		t.Error("hashing must NOT be paid for an already taken address")
	}
	if observed.called("SaveUser") {
		t.Error("no write must take place")
	}
}
