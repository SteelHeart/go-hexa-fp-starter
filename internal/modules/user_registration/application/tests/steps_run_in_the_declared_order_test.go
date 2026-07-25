package tests

import (
	"testing"
)

// TestStepsRunInTheDeclaredOrder : le pipeline s'exécute dans l'ordre où il se lit.
//
// C'est la promesse du style : le corps de `NewRegisterUser` se lit comme la liste
// des étapes métier, et cette lecture doit dire la vérité. Un ordre différent de
// celui du code rendrait le cas d'usage illisible — on croirait comprendre.
//
// Deux positions ont en plus une raison de SÉCURITÉ ou de COÛT :
//   - la disponibilité de l'adresse est vérifiée avant le hachage (coût) ;
//   - l'événement est enregistré après l'écriture (pas d'événement fantôme).
func TestStepsRunInTheDeclaredOrder(t *testing.T) {
	t.Parallel()

	observe := &journal{}
	_ = utilisateurDe(t, inscrit(depsNominales(observe)))

	attendu := []string{
		"EmailIsTaken",
		"HashPassword",
		"GenerateID",
		"Now",
		"SaveUser",
		"PublishEvent",
	}

	if len(observe.appels) != len(attendu) {
		t.Fatalf("appels = %v, attendu %v", observe.appels, attendu)
	}
	for i, nom := range attendu {
		if observe.appels[i] != nom {
			t.Fatalf("appels = %v, attendu %v", observe.appels, attendu)
		}
	}
}
