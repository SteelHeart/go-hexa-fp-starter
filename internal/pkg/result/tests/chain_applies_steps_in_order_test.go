package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestChainAppliesStepsInOrder : les étapes s'appliquent dans l'ordre déclaré, et
// chacune reçoit la sortie de la précédente.
//
// L'ordre est la seule chose qu'un lecteur déduit de la forme du code. S'il n'était
// pas garanti, lire un cas d'usage ne dirait plus ce qu'il fait.
func TestChainAppliesStepsInOrder(t *testing.T) {
	t.Parallel()

	sortie := result.Chain(okInt(1),
		func(n int) result.Result[int, erreur] { return okInt(double(n)) },     // 2
		func(n int) result.Result[int, erreur] { return okInt(incremente(n)) }, // 3
		func(n int) result.Result[int, erreur] { return okInt(double(n)) },     // 6
	)

	if !sortie.IsOk() {
		t.Fatalf("chaîne en échec: %q", cause(sortie))
	}
	if got := valeur(sortie); got != 6 {
		t.Errorf("valeur = %d, attendu 6 — les étapes ne s'enchaînent pas dans l'ordre", got)
	}
}
