package tests

import (
	"testing"
)

// TestSuccessfulRegistrationProducesAPendingUser: the nominal path, end to end,
// with no database and no container.
//
// This is what the purity of the core buys: this test runs in microseconds and
// yet covers the whole of the business rule. The same guarantees through an
// end-to-end test would cost a database, a broker, and several seconds.
func TestSuccessfulRegistrationProducesAPendingUser(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	user := userOf(t, register(nominalDeps(observed)))

	if user.ID != identifier {
		t.Errorf("identifier = %q, want %q", user.ID, identifier)
	}
	if user.Email.String() != validAddress {
		t.Errorf("address = %q, want %q", user.Email.String(), validAddress)
	}
	if user.PasswordHash.String() != digest {
		t.Errorf("digest = %q, want the one returned by the port", user.PasswordHash.String())
	}
	if !user.CreatedAt.Equal(fixedInstant()) {
		t.Errorf("creation date = %v, want the injected instant", user.CreatedAt)
	}
	if user.CanAuthenticate() {
		t.Error("a freshly registered account must not be able to sign in yet")
	}
}
