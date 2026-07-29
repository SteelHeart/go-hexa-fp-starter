package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestHashFailurePreventsPersistence: a failed hashing stops everything.
//
// The danger would be a user written to the database with an empty digest: they
// would exist, occupy their address, and never be able to sign in. Worse, an
// empty digest compared naively could let any password through, depending on the
// verification implementation.
//
// The short circuit makes that scenario structurally impossible.
func TestHashFailurePreventsPersistence(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	deps := nominalDeps(observed)
	deps.HashPassword = func(domain.RawPassword) result.Result[domain.PasswordHash, domain.Error] {
		observed.note("HashPassword")
		return failing[domain.PasswordHash](domain.CodeInternal, "hachage indisponible")
	}

	if got := codeOf(t, register(deps)); got != domain.CodeInternal {
		t.Errorf("code = %q, want %q", got, domain.CodeInternal)
	}
	if observed.called("SaveUser") {
		t.Error("no user must be written without a valid digest")
	}
	if observed.called("PublishEvent") {
		t.Error("no event must be published")
	}
}
