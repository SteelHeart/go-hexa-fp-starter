package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// TestEventNameCarriesItsVersion : le nom d'un événement porte sa version.
//
// Un consommateur est déployé INDÉPENDAMMENT du producteur. Sans version dans le
// nom, changer la forme de la charge casse silencieusement tous les consommateurs
// encore en vol — et le producteur, lui, ne voit aucune erreur.
//
// Avec la version, la v2 apparaît à côté de la v1 : les consommateurs migrent à
// leur rythme, et la v1 se retire quand plus personne ne l'écoute.
func TestEventNameCarriesItsVersion(t *testing.T) {
	t.Parallel()

	if !strings.HasSuffix(domain.EventUserRegistered, ".v1") {
		t.Errorf("nom = %q : un événement sans version est une rupture silencieuse en puissance",
			domain.EventUserRegistered)
	}
}
