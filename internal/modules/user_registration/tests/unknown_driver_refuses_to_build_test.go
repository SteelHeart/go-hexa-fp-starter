package tests

import (
	"strings"
	"testing"
	"time"

	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
)

// TestUnknownDriverRefusesToBuild : un pilote inconnu refuse le montage.
//
// Deny par défaut, et l'enjeu est concret : se rabattre sur le pilote par défaut
// serait BIEN PIRE que l'erreur. Une faute de frappe dans la configuration de
// production — `postgersql` au lieu de `postgres` — ferait démarrer le service
// en MÉMOIRE, sans rien signaler. Le service répondrait normalement, et la perte
// de toutes les inscriptions ne se verrait qu'au premier redémarrage.
func TestUnknownDriverRefusesToBuild(t *testing.T) {
	t.Parallel()

	publisher := &spyPublisher{}
	mod, err := userregistration.New("postgersql", userregistration.Deps{
		HashPassword: fakeHash,
		PublishEvent: publisher.port(),
		GenerateID:   sequentialIDs(),
		Now:          func() time.Time { return fixedInstant },
	})
	if err == nil {
		t.Fatal("un pilote inconnu doit refuser le montage")
	}

	// Le module rendu doit être inerte : si New échoue mais rend quand même un
	// module appelable, un appelant qui ignore l'erreur croirait fonctionner.
	if mod.Register != nil {
		t.Error("un montage refusé ne doit rendre aucun cas d'usage appelable")
	}
	if !strings.Contains(err.Error(), "postgersql") {
		t.Errorf("l'erreur doit nommer le pilote fautif, reçu: %v", err)
	}
}
