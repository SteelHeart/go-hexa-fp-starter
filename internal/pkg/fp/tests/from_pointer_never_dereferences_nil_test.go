package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/fp"
)

// TestFromPointerNeverDereferencesNil : FromPointer est le point de conversion des
// frontières.
//
// Au-delà, le domaine ne manipule plus de pointeur — donc plus aucun accès nil
// possible. Le test vérifie qu'elle ne déréférence pas nil : si elle le faisait,
// la conversion censée SUPPRIMER la classe de défauts en serait elle-même le
// dernier refuge.
func TestFromPointerNeverDereferencesNil(t *testing.T) {
	t.Parallel()

	var absent *string
	converti := fp.FromPointer(absent) // ne doit pas paniquer
	if converti.IsSome() {
		t.Error("un pointeur nil doit devenir None")
	}

	valeur := "présent"
	present := fp.FromPointer(&valeur)
	if !present.IsSome() {
		t.Fatal("un pointeur non nil doit devenir Some")
	}
	if got, _ := present.Get(); got != "présent" {
		t.Errorf("valeur = %q", got)
	}

	// La copie est faite : muter la source ne doit pas altérer l'Option.
	valeur = "muté"
	if got, _ := present.Get(); got != "présent" {
		t.Errorf("l'Option a suivi la mutation de la source: %q", got)
	}
}
