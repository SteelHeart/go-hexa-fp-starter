package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestStorageFailurePublishesNothing : une écriture en échec ne publie aucun
// événement.
//
// C'est la moitié du patron outbox. Publier « utilisateur inscrit » alors que
// l'écriture a échoué produirait un ÉVÉNEMENT FANTÔME : les consommateurs
// enverraient un courriel de bienvenue, créeraient un profil, factureraient — pour
// un utilisateur qui n'existe pas.
//
// Un événement fantôme est plus grave qu'un événement perdu : le perdu se rejoue,
// le fantôme se rattrape à la main, dans plusieurs systèmes.
func TestStorageFailurePublishesNothing(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	deps := depsNominales(observe)
	deps.SaveUser = func(context.Context, domain.User) result.Result[domain.User, domain.Error] {
		observe.note("SaveUser")
		return echoue[domain.User](domain.CodeUnavailable, "base injoignable")
	}

	if got := codeDe(t, inscrit(deps)); got != domain.CodeUnavailable {
		t.Errorf("code = %q, attendu %q", got, domain.CodeUnavailable)
	}
	if observe.aAppele("PublishEvent") {
		t.Error("aucun événement ne doit être publié quand l'écriture a échoué")
	}
}
