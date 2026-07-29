package tests

import (
	"strings"
	"testing"
)

// TestTheClearPasswordNeverReachesStorage is the security test of the use case.
//
// The clear-text password must reach ONLY ONE port: the hashing one. Neither the
// write nor the event must see it — because what reaches the storage ends up in
// the backups, and what reaches the event ends up in the broker, then at every
// consumer.
//
// The `RawPassword` type protects against leaks through logging; this test
// protects against the leak through TRANSPORT, which the type cannot prevent.
func TestTheClearPasswordNeverReachesStorage(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	user := userOf(t, register(nominalDeps(observed)))

	// The hashing port, for its part, must indeed receive the value: that is
	// its role.
	if observed.hashed != strongPassword {
		t.Error("the hashing port must receive the clear-text password")
	}

	if observed.saved.PasswordHash.String() != digest {
		t.Errorf("the written user carries %q, want the digest",
			observed.saved.PasswordHash.String())
	}
	if strings.Contains(observed.saved.PasswordHash.String(), strongPassword) {
		t.Error("the clear-text password reached the storage")
	}
	if user.PasswordHash.String() == strongPassword {
		t.Error("the returned user carries the clear-text password")
	}
}
