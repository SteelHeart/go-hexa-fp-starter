package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestDisabledModuleRefusesAuthorizeToo garde la pire valeur par défaut imaginable.
//
// # Le cas qu'on oublie
//
// Un module désactivé se monte quand même — c'est ce qui permet à une surface
// d'exister et de répondre une erreur claire, plutôt que de faire échouer le
// démarrage entier pour un module que personne n'a activé.
//
// La tentation est alors d'écrire les refus « qui comptent » — s'inscrire, se
// connecter — et de laisser `Authorize` à sa valeur zéro. Or le zéro d'un port
// fonction est `nil`, et un `nil` appelé panique ; pire, une implémentation
// « neutre » qui rendrait `nil` en erreur AUTORISERAIT tout. Un module
// d'authentification éteint qui autorise tout est exactement l'inverse du deny
// par défaut.
func TestDisabledModuleRefusesAuthorizeToo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, err := auth.New(config.Module{Enabled: false}, auth.Deps{})
	if err != nil {
		t.Fatalf("un module désactivé doit se monter : %v", err)
	}

	if err := mod.Authorize(ctx, "n-importe-qui", permission(t, "billing.invoice.cancel")); !errors.Is(err, auth.ErrDisabled) {
		t.Fatalf("Authorize sur module désactivé : attendu ErrDisabled, obtenu %v", err)
	}

	if _, err := mod.Register(ctx, subject, secret); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("Register : attendu ErrDisabled, obtenu %v", err)
	}
	if _, err := mod.Authenticate(ctx, subject, secret); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("Authenticate : attendu ErrDisabled, obtenu %v", err)
	}
	if _, err := mod.Verify(ctx, domain.Token{}); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("Verify : attendu ErrDisabled, obtenu %v", err)
	}
	if err := mod.Revoke(ctx, domain.Token{}); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("Revoke : attendu ErrDisabled, obtenu %v", err)
	}
	if err := mod.DefineRole(ctx, "comptable", nil); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("DefineRole : attendu ErrDisabled, obtenu %v", err)
	}
	if err := mod.AssignRoles(ctx, "n-importe-qui", nil); !errors.Is(err, auth.ErrDisabled) {
		t.Errorf("AssignRoles : attendu ErrDisabled, obtenu %v", err)
	}
}
