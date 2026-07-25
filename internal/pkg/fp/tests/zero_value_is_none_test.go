package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestZeroValueIsNone : la valeur zéro d'une Option est None.
//
// C'est la même propriété que pour Result, et pour la même raison : un champ
// Option oublié doit valoir « absent », jamais « présent et vide ». Si la valeur
// zéro était un Some portant le zéro de T, un utilisateur sans date de naissance
// paraîtrait né le 1ᵉʳ janvier de l'an 1.
func TestZeroValueIsNone(t *testing.T) {
	t.Parallel()

	var oubliee fp.Option[string]

	if oubliee.IsSome() {
		t.Fatal("la valeur zéro d'une Option doit être None")
	}
	if !oubliee.IsNone() {
		t.Error("IsNone doit être vrai sur la valeur zéro")
	}
	if _, present := oubliee.Get(); present {
		t.Error("Get doit rendre present=false sur la valeur zéro")
	}
	if got := oubliee.ValueOr("repli"); got != "repli" {
		t.Errorf("ValueOr = %q, attendu le repli", got)
	}
}
