package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestLockKeyDistinguishesTasks: two distinct names give two distinct keys.
//
// A collision would make two unrelated tasks exclude each other. That is
// improbable over 63 bits, and the worst case remains a serialised execution —
// never a double execution — but verifying it on the names actually in use
// costs three lines and saves having to argue about it.
func TestLockKeyDistinguishesTasks(t *testing.T) {
	t.Parallel()

	seen := map[int64]domain.TaskName{}
	for _, name := range []domain.TaskName{
		"purge-idempotency", "purge-outbox", "reminders", "billing", "accounting-export",
	} {
		key := domain.LockKey(name)
		if other, clash := seen[key]; clash {
			t.Errorf("collision between %q and %q on the key %d", name, other, key)
		}
		seen[key] = name
	}
}
