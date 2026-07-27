package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/modulebus"
)

// TestADisabledModuleFailsAtCallNotAtMount : désactivé se résout, mais n'appelle pas.
//
// # Pourquoi cette asymétrie est voulue
//
// `disabled` est le seul mode dont la résolution RÉUSSIT alors qu'aucun appel ne
// pourra aboutir. Ce n'est pas une incohérence, c'est le but : le binaire doit
// démarrer avec une capacité coupée — pendant un incident chez le fournisseur,
// pendant une migration — sans qu'il faille retirer du code.
//
// Mais la coupure doit être BRUYANTE à l'usage. Le piège serait de rendre la
// valeur zéro et `nil` : l'appelant lirait un enregistrement vide comme une
// réponse valide, et traiterait « capacité coupée » comme « rien à faire ». Une
// vérification d'éligibilité désactivée rendrait alors « non éligible » sans que
// personne ne s'en aperçoive.
//
// Deny par défaut jusqu'ici : coupé signifie refusé, jamais vide.
func TestADisabledModuleFailsAtCallNotAtMount(t *testing.T) {
	t.Parallel()

	var localCalls int
	call := resolve(t, interop("disabled", nil), noPublisher(t), localCaller(&localCalls))

	got, err := call(context.Background(), request{Ref: "r-1"})

	if err == nil {
		t.Fatal("un module désactivé a rendu nil — l'appelant lirait la valeur zéro " +
			"comme une réponse valide")
	}
	if !errors.Is(err, modulebus.ErrDisabled) {
		t.Errorf("erreur rendue = %v, attendu ErrDisabled", err)
	}
	if got.Accepted {
		t.Errorf("réponse non nulle rendue avec l'erreur: %+v", got)
	}
	if localCalls != 0 {
		t.Errorf("l'implémentation locale a été appelée %d fois — "+
			"« désactivé » ne doit pas se rabattre sur l'appel local", localCalls)
	}
}
