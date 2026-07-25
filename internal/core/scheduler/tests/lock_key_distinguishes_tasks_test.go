package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// TestLockKeyDistinguishesTasks : deux noms distincts donnent deux clés distinctes.
//
// Une collision ferait s'exclure mutuellement deux tâches sans lien. C'est
// improbable sur 63 bits, et le pire cas reste une exécution sérialisée — jamais une
// double exécution — mais le vérifier sur les noms réellement utilisés coûte trois
// lignes et évite d'en discuter.
func TestLockKeyDistinguishesTasks(t *testing.T) {
	t.Parallel()

	seen := map[int64]domain.TaskName{}
	for _, name := range []domain.TaskName{
		"purge-idempotency", "purge-outbox", "rappels", "facturation", "export-comptable",
	} {
		key := domain.LockKey(name)
		if other, clash := seen[key]; clash {
			t.Errorf("collision entre %q et %q sur la clé %d", name, other, key)
		}
		seen[key] = name
	}
}
