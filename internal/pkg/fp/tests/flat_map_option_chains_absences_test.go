package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestFlatMapOptionChainsAbsences : enchaîner des recherches qui peuvent ne rien
// trouver ne doit pas produire une Option d'Option.
//
// C'est toute la différence avec MapOption : chercher l'employeur d'un utilisateur
// dont on n'est pas sûr qu'il existe rend « pas d'employeur », pas « peut-être un
// peut-être ».
func TestFlatMapOptionChainsAbsences(t *testing.T) {
	t.Parallel()

	positifOuRien := func(n int) fp.Option[int] {
		if n <= 0 {
			return fp.None[int]()
		}
		return fp.Some(n)
	}

	if got := fp.FlatMapOption(fp.Some(5), positifOuRien); !got.IsSome() {
		t.Error("une valeur valide doit traverser")
	}
	if got := fp.FlatMapOption(fp.Some(-5), positifOuRien); got.IsSome() {
		t.Error("une valeur refusée par f doit donner None")
	}

	appelee := false
	absent := fp.FlatMapOption(fp.None[int](), func(n int) fp.Option[int] {
		appelee = true
		return fp.Some(n)
	})
	if appelee {
		t.Error("f ne doit PAS être appelée sur une Option vide")
	}
	if absent.IsSome() {
		t.Error("None doit traverser inchangée")
	}
}
