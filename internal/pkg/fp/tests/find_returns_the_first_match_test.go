package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestFindReturnsTheFirstMatch : Find rend le PREMIER élément satisfaisant le
// prédicat, et None si aucun ne le satisfait.
//
// « Premier » est le contrat, pas un hasard d'implémentation : sur une liste
// ordonnée — la plus récente en tête — rendre le dernier au lieu du premier
// inverserait le sens de la recherche sans qu'aucun type ne s'y oppose.
//
// L'absence rend None plutôt qu'une valeur zéro : c'est ce qui distingue
// « aucun résultat » de « un résultat vide ».
func TestFindReturnsTheFirstMatch(t *testing.T) {
	t.Parallel()

	trouve := fp.Find([]int{1, 3, 4, 6, 8}, pair)
	if !trouve.IsSome() {
		t.Fatal("un élément satisfait le prédicat : Find doit le trouver")
	}
	if got, _ := trouve.Get(); got != 4 {
		t.Errorf("Find = %d, attendu 4 — le PREMIER pair, pas un autre", got)
	}

	if absent := fp.Find([]int{1, 3, 5}, pair); absent.IsSome() {
		t.Error("aucun élément ne satisfait le prédicat : Find doit rendre None")
	}
}
