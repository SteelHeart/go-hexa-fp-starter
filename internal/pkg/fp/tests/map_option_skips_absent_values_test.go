package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestMapOptionSkipsAbsentValues : la transformation n'est PAS appelée sur une
// Option vide.
//
// C'est ce qui permet d'enchaîner des transformations sans tester la présence à
// chaque étape. Si la fonction était appelée sur l'absence, elle recevrait la
// valeur zéro de T et produirait un résultat qui a l'air valide.
func TestMapOptionSkipsAbsentValues(t *testing.T) {
	t.Parallel()

	appelee := false
	sortie := fp.MapOption(fp.None[int](), func(n int) string {
		appelee = true
		return versTexte(n)
	})

	if appelee {
		t.Error("la transformation ne doit PAS être appelée sur une Option vide")
	}
	if sortie.IsSome() {
		t.Error("MapOption sur None doit rendre None")
	}
}
