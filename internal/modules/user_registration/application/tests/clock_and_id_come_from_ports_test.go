package tests

import (
	"testing"
)

// TestClockAndIDComeFromPorts: neither the time nor the identifier is produced
// by the core.
//
// That is what makes this test deterministic, and it is also what makes the use
// case usable by a seeder: replaying a registration with a chosen identifier and
// date allows a reproducible data set to be rebuilt.
//
// A `time.Now()` or a `uuid.New()` slipped into the pipeline would break both
// properties at once, and the test would fail — which is exactly its role.
func TestClockAndIDComeFromPorts(t *testing.T) {
	t.Parallel()

	observed := &callLog{}
	user := userOf(t, register(nominalDeps(observed)))

	if !observed.called("Now") {
		t.Error("the Now port must be called: the core does not read the clock")
	}
	if !observed.called("GenerateID") {
		t.Error("the GenerateID port must be called: the core does not produce randomness")
	}
	if !user.CreatedAt.Equal(fixedInstant()) {
		t.Errorf("date = %v: it does not come from the port", user.CreatedAt)
	}
}
