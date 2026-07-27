package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestTraverseStopsAtFirstError : Traverse applique une fonction faillible à chaque
// élément et s'arrête à la première erreur — SANS appeler la fonction sur les
// éléments suivants.
//
// L'arrêt effectif compte autant que le résultat : si la fonction produit un effet
// — écrire en base, appeler un service — continuer après une erreur laisserait un
// état partiel que personne n'a demandé.
func TestTraverseStopsAtFirstError(t *testing.T) {
	t.Parallel()

	var vus []int
	valide := func(n int) result.Result[string, erreur] {
		vus = append(vus, n)
		if n < 0 {
			return result.Err[string, erreur]("valeur négative")
		}
		return result.Ok[string, erreur](versTexte(n))
	}

	tous := result.Traverse([]int{1, 2, 3}, valide)
	if !tous.IsOk() {
		t.Fatalf("aucune erreur attendue, reçu %q", cause(tous))
	}
	if got := valeur(tous); len(got) != 3 || got[2] != "3" {
		t.Errorf("valeurs = %v, attendu [1 2 3] en texte", got)
	}

	vus = nil
	partiel := result.Traverse([]int{1, -2, 3}, valide)
	if partiel.IsOk() {
		t.Fatal("un élément invalide doit faire échouer tout le parcours")
	}
	if len(vus) != 2 {
		t.Errorf("éléments parcourus = %v, attendu l'arrêt au deuxième", vus)
	}
}
