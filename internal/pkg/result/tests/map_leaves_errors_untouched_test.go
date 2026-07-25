package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// TestMapLeavesErrorsUntouched : un Result en erreur traverse Map sans que la
// transformation soit appelée.
//
// C'est le court-circuit qui rend un pipeline lisible : les étapes suivantes n'ont
// pas à vérifier si les précédentes ont réussi. Si la fonction était appelée quand
// même, chaque étape devrait se protéger elle-même et tout le gain disparaîtrait.
func TestMapLeavesErrorsUntouched(t *testing.T) {
	t.Parallel()

	appelee := false
	transforme := func(n int) string {
		appelee = true
		return versTexte(n)
	}

	sortie := result.Map(errInt("refusé"), transforme)

	if appelee {
		t.Error("la transformation ne doit PAS être appelée sur un Result en erreur")
	}
	if sortie.IsOk() {
		t.Error("Map sur une erreur doit rendre une erreur")
	}
	if cause(sortie) != "refusé" {
		t.Errorf("l'erreur doit traverser inchangée, reçu %q", cause(sortie))
	}
}
