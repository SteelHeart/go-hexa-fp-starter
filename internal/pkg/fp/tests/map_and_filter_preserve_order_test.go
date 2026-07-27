package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestMapAndFilterPreserveOrder : l'ordre des éléments est conservé.
//
// Une pagination par curseur repose entièrement sur un ordre stable : une fonction
// qui réordonnerait silencieusement ferait sauter ou répéter des lignes entre deux
// pages, et le symptôme n'apparaîtrait qu'au-delà de la première page.
func TestMapAndFilterPreserveOrder(t *testing.T) {
	t.Parallel()

	source := []int{3, 1, 4, 1, 5}

	transforme := fp.Map(source, double)
	attendu := []int{6, 2, 8, 2, 10}
	for i, want := range attendu {
		if transforme[i] != want {
			t.Fatalf("Map = %v, attendu %v", transforme, attendu)
		}
	}

	garde := fp.Filter([]int{1, 2, 3, 4, 5, 6}, pair)
	attenduFiltre := []int{2, 4, 6}
	if len(garde) != len(attenduFiltre) {
		t.Fatalf("Filter = %v, attendu %v", garde, attenduFiltre)
	}
	for i, want := range attenduFiltre {
		if garde[i] != want {
			t.Fatalf("Filter = %v, attendu %v", garde, attenduFiltre)
		}
	}
}
