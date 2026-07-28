package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestUnknownSubjectAndWrongSecretAreIndistinguishable ferme l'oracle d'existence
// de comptes.
//
// # Ce que la faute produit quand elle est là
//
// Un formulaire qui répond « cet identifiant n'existe pas » d'un côté et « mot de
// passe incorrect » de l'autre est un service d'énumération : on lui soumet mille
// adresses, on note lesquelles répondent différemment, et on sait exactement où
// concentrer l'effort. C'est la faute la plus répandue du domaine, et elle est
// invisible à la relecture parce que chaque message est, isolément, plus utile.
//
// Les trois causes — sujet mal formé, sujet inconnu, secret faux — doivent rendre
// la MÊME erreur. Le test compare les trois entre elles, pas à une valeur
// attendue : c'est l'indiscernabilité qui est la propriété, pas le libellé.
func TestUnknownSubjectAndWrongSecretAreIndistinguishable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	register(t, mod, subject)

	cases := map[string]struct{ subject, secret string }{
		"sujet inconnu":   {"personne@example.com", secret},
		"secret faux":     {subject, "un-autre-secret-long"},
		"sujet mal formé": {"   ", secret},
		"secret vide":     {subject, ""},
	}

	messages := make(map[string]bool)
	for name, tc := range cases {
		_, err := mod.Authenticate(ctx, tc.subject, tc.secret)
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("%s : attendu ErrInvalidCredentials, obtenu %v", name, err)
		}
		messages[err.Error()] = true
	}

	if len(messages) != 1 {
		t.Fatalf("les refus doivent être indiscernables ; %d messages distincts : %v",
			len(messages), messages)
	}
}
