package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestMapOptionObeysFunctorLaws : identité et composition, pour les mêmes raisons
// que sur Result.
//
// Sans la loi de composition, fusionner deux transformations successives en une
// seule changerait le résultat — et ce genre de regroupement se fait sans y penser
// lors d'un nettoyage de code.
func TestMapOptionObeysFunctorLaws(t *testing.T) {
	t.Parallel()

	identite := func(n int) int { return n }

	for _, o := range []fp.Option[int]{fp.Some(21), fp.None[int]()} {
		applique := fp.MapOption(o, identite)
		gotV, gotP := applique.Get()
		wantV, wantP := o.Get()
		if gotV != wantV || gotP != wantP {
			t.Error("loi d'identité rompue")
		}
	}

	depart := fp.Some(10)
	enDeux := fp.MapOption(fp.MapOption(depart, double), incremente)
	enUne := fp.MapOption(depart, func(n int) int { return incremente(double(n)) })

	if enDeux.ValueOr(-1) != enUne.ValueOr(-1) {
		t.Errorf("loi de composition rompue: %d ≠ %d", enDeux.ValueOr(-1), enUne.ValueOr(-1))
	}
}
