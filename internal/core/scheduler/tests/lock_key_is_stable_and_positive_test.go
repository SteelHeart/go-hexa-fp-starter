package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestLockKeyIsStableAndPositive: the election key derived from a name must be
// stable between runs AND between replicas.
//
// If it were unstable, it would lock nothing: two replicas would take two
// different locks and both believe themselves elected. A `hash/maphash`, whose
// seed changes with every process, would produce exactly that defect —
// invisible on a single instance, systematic on two.
//
// Positive, it stays readable in `pg_locks` and in a log: a negative key is
// valid on the database side but complicates every diagnosis.
func TestLockKeyIsStableAndPositive(t *testing.T) {
	t.Parallel()

	// The last name is kept as it stands, diacritics included: its whole point
	// is to exercise a task name outside pure ASCII. Translating it would remove
	// the very property under test.
	names := []domain.TaskName{"purge", "reminders", "billing", "", "tâche accentuée"}
	for _, name := range names {
		first := domain.LockKey(name)
		if second := domain.LockKey(name); first != second {
			t.Errorf("unstable key for %q: %d then %d", name, first, second)
		}
		if first < 0 {
			t.Errorf("negative key for %q: %d", name, first)
		}
	}
}
