package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestDefaultDriverNeedsNoInfrastructure : le module se monte et INSCRIT sans
// base, sans cache, sans conteneur.
//
// C'est la promesse commerciale du socle, pas une commodité de test : `hexa new`
// puis `go run` doit répondre sur une machine vierge. Un module métier dont le
// seul pilote exigerait PostgreSQL la briserait au premier module écrit —
// c'est-à-dire au moment exact où un évaluateur l'éprouve.
//
// Le test vérifie le chemin COMPLET, pas seulement le montage : un module qui se
// construit puis échoue à la première écriture ne prouverait rien.
func TestDefaultDriverNeedsNoInfrastructure(t *testing.T) {
	t.Parallel()

	publisher := &spyPublisher{}
	mod := newModule(t, publisher)

	user := register(t, mod, "alice@example.com", validPassword)

	if user.ID.IsZero() {
		t.Error("l'utilisateur inscrit doit porter un identifiant")
	}
	if user.Status != domain.StatusPending {
		t.Errorf("statut = %q, attendu %q — un compte ne naît jamais actif",
			user.Status, domain.StatusPending)
	}
	if !user.CreatedAt.Equal(fixedInstant) {
		t.Errorf("créé le %v, attendu %v — l'horloge doit venir du port",
			user.CreatedAt, fixedInstant)
	}
}
