package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestCollectStopsAtFirstError : Collect transforme une liste de Result en un
// Result de liste, et s'arrête à la première erreur.
//
// « Tout ou rien » est le bon contrat ici : rendre une liste partielle obligerait
// chaque appelant à décider quoi faire des éléments manquants, et le premier qui
// oublierait de le faire traiterait un lot incomplet comme un lot complet.
func TestCollectStopsAtFirstError(t *testing.T) {
	t.Parallel()

	tous := result.Collect([]result.Result[int, erreur]{okInt(1), okInt(2), okInt(3)})
	if !tous.IsOk() {
		t.Fatalf("une liste de succès doit rendre un succès, reçu %q", cause(tous))
	}
	if got := valeur(tous); len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("valeurs rassemblées = %v, attendu [1 2 3]", got)
	}

	partiel := result.Collect([]result.Result[int, erreur]{
		okInt(1), errInt("deuxième refusé"), errInt("troisième refusé"),
	})
	if partiel.IsOk() {
		t.Fatal("une seule erreur doit faire échouer tout le lot")
	}
	if cause(partiel) != "deuxième refusé" {
		t.Errorf("erreur = %q, attendu la PREMIÈRE rencontrée", cause(partiel))
	}
}
