package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestMapObeysFunctorLaws vérifie les deux lois qui rendent le refactoring sûr.
//
//  1. IDENTITÉ : Map(r, x => x) == r. Une transformation qui ne transforme rien ne
//     doit rien changer — sinon insérer une étape neutre dans un pipeline en
//     changerait le résultat.
//  2. COMPOSITION : Map(Map(r, f), g) == Map(r, x => g(f(x))). C'est la loi qui
//     autorise à FUSIONNER deux étapes en une, ou à en extraire une troisième,
//     sans rien casser. Sans elle, tout regroupement d'étapes serait un pari.
func TestMapObeysFunctorLaws(t *testing.T) {
	t.Parallel()

	identite := func(n int) int { return n }

	for _, r := range []result.Result[int, erreur]{okInt(21), errInt("refusé")} {
		applique := result.Map(r, identite)
		if applique.IsOk() != r.IsOk() ||
			valeur(applique) != valeur(r) ||
			cause(applique) != cause(r) {
			t.Error("loi d'identité rompue")
		}
	}

	depart := okInt(10)
	enDeuxEtapes := result.Map(result.Map(depart, double), incremente)
	enUneEtape := result.Map(depart, func(n int) int { return incremente(double(n)) })

	if valeur(enDeuxEtapes) != valeur(enUneEtape) {
		t.Errorf("loi de composition rompue: %d ≠ %d",
			valeur(enDeuxEtapes), valeur(enUneEtape))
	}
}
