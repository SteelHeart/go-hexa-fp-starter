package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestSomeOfZeroIsNotNone : `Some("")` est PRÉSENT, et distinct de None.
//
// C'est la distinction que le pointeur nil ne sait pas faire et que l'Option
// existe pour rendre : « champ jamais renseigné » et « champ volontairement vidé »
// sont deux décisions différentes. Les confondre fait réapparaître une valeur par
// défaut là où quelqu'un avait explicitement demandé le vide.
func TestSomeOfZeroIsNotNone(t *testing.T) {
	t.Parallel()

	vide := fp.Some("")
	if !vide.IsSome() {
		t.Fatal("Some(\"\") doit être présent")
	}
	if got := vide.ValueOr("repli"); got != "" {
		t.Errorf("ValueOr = %q : le repli a écrasé une valeur vide DÉLIBÉRÉE", got)
	}

	zero := fp.Some(0)
	if !zero.IsSome() || zero.ValueOr(42) != 0 {
		t.Error("Some(0) doit être présent et rendre 0, pas le repli")
	}
}
