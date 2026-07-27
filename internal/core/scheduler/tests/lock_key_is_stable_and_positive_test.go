package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestLockKeyIsStableAndPositive : la clé d'élection dérivée d'un nom doit être
// stable entre exécutions ET entre répliques.
//
// Instable, elle ne verrouillerait rien : deux répliques prendraient deux verrous
// différents et s'estimeraient toutes les deux élues. Un `hash/maphash`, dont la
// graine change à chaque processus, produirait exactement ce défaut — invisible sur
// une seule instance, systématique sur deux.
//
// Positive, elle reste lisible dans `pg_locks` et dans un journal : une clé négative
// est valide côté base mais complique tout diagnostic.
func TestLockKeyIsStableAndPositive(t *testing.T) {
	t.Parallel()

	names := []domain.TaskName{"purge", "rappels", "facturation", "", "tâche accentuée"}
	for _, name := range names {
		first := domain.LockKey(name)
		if second := domain.LockKey(name); first != second {
			t.Errorf("clé instable pour %q: %d puis %d", name, first, second)
		}
		if first < 0 {
			t.Errorf("clé négative pour %q: %d", name, first)
		}
	}
}
