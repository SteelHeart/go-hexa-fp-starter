package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestReduceFoldsInOrder : Reduce replie de gauche à droite.
//
// L'ordre est invisible sur une addition et décisif sur tout le reste. Une
// concaténation repliée à l'envers rend une chaîne inversée — un résultat qui a
// l'air correct jusqu'à ce qu'on le lise.
func TestReduceFoldsInOrder(t *testing.T) {
	t.Parallel()

	somme := fp.Reduce([]int{1, 2, 3, 4}, 0, func(acc, n int) int { return acc + n })
	if somme != 10 {
		t.Errorf("somme = %d, attendu 10", somme)
	}

	concat := fp.Reduce([]string{"a", "b", "c"}, "", func(acc, s string) string { return acc + s })
	if concat != "abc" {
		t.Errorf("concaténation = %q, attendu \"abc\" — le repli va de gauche à droite", concat)
	}
}
