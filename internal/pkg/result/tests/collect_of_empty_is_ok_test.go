package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestCollectOfEmptyIsOk : une liste vide rassemble en un SUCCÈS portant une liste
// vide.
//
// Ce n'est pas une évidence : la valeur zéro d'un Result étant un Err, il serait
// facile de rendre une erreur par accident. Or « rien à traiter » n'est pas un
// échec, et une page de résultats vide n'est pas une panne.
func TestCollectOfEmptyIsOk(t *testing.T) {
	t.Parallel()

	vide := result.Collect([]result.Result[int, erreur]{})
	if !vide.IsOk() {
		t.Fatalf("une liste vide doit rendre un succès, reçu %q", cause(vide))
	}
	if got := valeur(vide); len(got) != 0 {
		t.Errorf("valeurs = %v, attendu une liste vide", got)
	}

	absente := result.Collect[int, erreur](nil)
	if !absente.IsOk() {
		t.Error("une liste nil doit se comporter comme une liste vide")
	}
}
