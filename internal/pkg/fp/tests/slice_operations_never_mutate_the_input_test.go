package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestSliceOperationsNeverMutateTheInput est la garantie centrale de ces trois
// fonctions.
//
// En Go, une tranche partage son tableau sous-jacent : une implémentation qui
// écrirait dans `items` modifierait la tranche de l'APPELANT, à distance et sans
// trace. C'est la classe de défauts la plus pénible à diagnostiquer, parce que le
// coupable est une fonction qui a l'air pure.
func TestSliceOperationsNeverMutateTheInput(t *testing.T) {
	t.Parallel()

	source := []int{1, 2, 3, 4}
	temoin := []int{1, 2, 3, 4}

	_ = fp.Map(source, double)
	_ = fp.Filter(source, pair)
	_ = fp.Reduce(source, 0, func(acc, n int) int { return acc + n })
	_ = fp.Find(source, pair)

	for i := range temoin {
		if source[i] != temoin[i] {
			t.Fatalf("l'entrée a été modifiée: %v, attendu %v", source, temoin)
		}
	}
}
