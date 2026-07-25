package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestEmptySlicesAreNotASpecialCase : une tranche vide OU nil traverse sans
// panique et rend une tranche vide, jamais nil.
//
// Rendre nil obligerait chaque appelant à distinguer « vide » de « nil » — une
// distinction que Go rend presque invisible et qui ressurgit à la sérialisation :
// `null` au lieu de `[]` dans une réponse JSON casse les clients typés.
func TestEmptySlicesAreNotASpecialCase(t *testing.T) {
	t.Parallel()

	for name, source := range map[string][]int{"vide": {}, "nil": nil} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := fp.Map(source, double); got == nil {
				t.Error("Map doit rendre une tranche vide, jamais nil")
			} else if len(got) != 0 {
				t.Errorf("Map = %v, attendu vide", got)
			}

			if got := fp.Filter(source, pair); got == nil {
				t.Error("Filter doit rendre une tranche vide, jamais nil")
			} else if len(got) != 0 {
				t.Errorf("Filter = %v, attendu vide", got)
			}

			if got := fp.Reduce(source, 100, func(acc, n int) int { return acc + n }); got != 100 {
				t.Errorf("Reduce = %d, attendu la valeur initiale intacte", got)
			}

			if got := fp.Find(source, pair); got.IsSome() {
				t.Error("Find dans une tranche vide doit rendre None")
			}
		})
	}
}
