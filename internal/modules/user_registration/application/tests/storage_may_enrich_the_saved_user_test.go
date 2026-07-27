package tests

import (
	"context"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestStorageMayEnrichTheSavedUser : c'est l'utilisateur RENDU par le stockage qui
// poursuit le pipeline, pas celui qu'on lui a donné.
//
// La différence compte dès que la base a le dernier mot sur une valeur : un
// identifiant produit par une séquence, un horodatage posé par un `DEFAULT now()`,
// un état ajusté par un déclencheur. Ignorer le retour publierait un événement
// portant des valeurs que la base n'a jamais écrites.
func TestStorageMayEnrichTheSavedUser(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	deps := depsNominales(observe)
	deps.SaveUser = func(_ context.Context, u domain.User) result.Result[domain.User, domain.Error] {
		observe.note("SaveUser")
		enrichi := u.WithStatus(domain.StatusActive) // la base a tranché autrement
		observe.sauvegarde = enrichi
		return result.Ok[domain.User, domain.Error](enrichi)
	}

	user := utilisateurDe(t, inscrit(deps))

	if user.Status != domain.StatusActive {
		t.Errorf("état = %q : le retour du stockage a été ignoré", user.Status)
	}
}
