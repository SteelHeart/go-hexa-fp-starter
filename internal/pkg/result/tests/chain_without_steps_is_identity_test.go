package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestChainWithoutStepsIsIdentity : une chaîne sans étape rend son entrée telle
// quelle, sur les deux branches.
//
// Cas limite qui a l'air théorique et ne l'est pas : une liste d'étapes construite
// dynamiquement — filtrée par un drapeau de fonctionnalité, par exemple — peut être
// vide. Elle doit alors être neutre, pas transformer un succès en autre chose.
func TestChainWithoutStepsIsIdentity(t *testing.T) {
	t.Parallel()

	for _, depart := range []result.Result[int, erreur]{okInt(7), errInt("refusé")} {
		sortie := result.Chain(depart)
		if sortie.IsOk() != depart.IsOk() ||
			valeur(sortie) != valeur(depart) ||
			cause(sortie) != cause(depart) {
			t.Error("une chaîne vide doit être neutre")
		}
	}
}
