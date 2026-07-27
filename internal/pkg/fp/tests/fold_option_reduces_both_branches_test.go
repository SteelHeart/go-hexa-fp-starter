package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestFoldOptionReducesBothBranches : FoldOption oblige à traiter l'absence.
//
// Les deux fonctions sont exigées à l'appel : impossible de sortir d'une Option en
// ayant oublié le cas vide. C'est la garantie que le compilateur apporte ici et
// qu'un pointeur nil n'apporte jamais.
func TestFoldOptionReducesBothBranches(t *testing.T) {
	t.Parallel()

	onSome := func(n int) string { return "présent:" + versTexte(n) }
	onNone := func() string { return "absent" }

	if got := fp.FoldOption(fp.Some(7), onSome, onNone); got != "présent:7" {
		t.Errorf("FoldOption sur Some = %q", got)
	}
	if got := fp.FoldOption(fp.None[int](), onSome, onNone); got != "absent" {
		t.Errorf("FoldOption sur None = %q", got)
	}
}
