package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestDefaultDriverNeedsNoInfrastructure: the module mounts and REGISTERS with
// no database, no cache, no container.
//
// This is the commercial promise of the starter, not a test convenience: `hexa
// new` then `go run` must answer on a blank machine. A business module whose
// only driver required PostgreSQL would break it on the first module written —
// that is to say at the exact moment an evaluator exercises it.
//
// The test verifies the COMPLETE path, not only the assembly: a module that
// builds then fails on the first write would prove nothing.
func TestDefaultDriverNeedsNoInfrastructure(t *testing.T) {
	t.Parallel()

	publisher := &spyPublisher{}
	mod := newModule(t, publisher)

	user := register(t, mod, "alice@example.com", validPassword)

	if user.ID.IsZero() {
		t.Error("the registered user must carry an identifier")
	}
	if user.Status != domain.StatusPending {
		t.Errorf("status = %q, want %q — an account is never born active",
			user.Status, domain.StatusPending)
	}
	if !user.CreatedAt.Equal(fixedInstant) {
		t.Errorf("created at %v, want %v — the clock must come from the port",
			user.CreatedAt, fixedInstant)
	}
}
