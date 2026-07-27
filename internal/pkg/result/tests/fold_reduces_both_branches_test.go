package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestFoldReducesBothBranches : Fold est la sortie canonique d'un Result dans un
// adaptateur primaire.
//
// Elle oblige à traiter les DEUX branches : impossible de rendre une réponse HTTP
// en ayant oublié le cas d'erreur, puisque les deux fonctions sont exigées à
// l'appel. Une seule branche doit être empruntée, jamais les deux.
func TestFoldReducesBothBranches(t *testing.T) {
	t.Parallel()

	var empruntees []string
	onOk := func(n int) string {
		empruntees = append(empruntees, "ok")
		return "succès:" + versTexte(n)
	}
	onErr := func(e erreur) string {
		empruntees = append(empruntees, "err")
		return "échec:" + string(e)
	}

	if got := result.Fold(okInt(7), onOk, onErr); got != "succès:7" {
		t.Errorf("Fold sur un succès = %q", got)
	}
	if got := result.Fold(errInt("refusé"), onOk, onErr); got != "échec:refusé" {
		t.Errorf("Fold sur une erreur = %q", got)
	}
	if len(empruntees) != 2 {
		t.Errorf("branches empruntées = %v, attendu exactement une par appel", empruntees)
	}
}
