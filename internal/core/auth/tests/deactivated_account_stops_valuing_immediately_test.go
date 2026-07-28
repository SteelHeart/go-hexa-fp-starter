package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestDeactivatedAccountStopsValuingImmediately couvre les TROIS portes à la fois.
//
// # Pourquoi les trois, et pas une
//
// `Active` est consulté à trois endroits — l'authentification, la résolution d'un
// jeton, et l'autorisation. N'en tester qu'un laisserait les deux autres dériver,
// et la faute serait la pire de sa catégorie : un compte affiché comme fermé qui
// continue de fonctionner par un chemin qu'on n'a pas regardé.
//
// Le jeton est émis AVANT la fermeture, exprès : c'est un jeton déjà en
// circulation, celui qu'un attaquant détient au moment où l'on réagit. Aucune
// expiration n'intervient.
func TestDeactivatedAccountStopsValuingImmediately(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	id := register(t, mod, subject)
	cancel := permission(t, "billing.invoice.cancel")
	grant(t, mod, id, "comptable", "billing.invoice.cancel")

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentification: %v", err)
	}

	if err := mod.Deactivate(ctx, id); err != nil {
		t.Fatalf("fermeture du compte: %v", err)
	}

	if _, err := mod.Authenticate(ctx, subject, secret); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("compte fermé, authentification : attendu ErrInvalidCredentials, obtenu %v", err)
	}
	if _, err := mod.Verify(ctx, session.Token); !errors.Is(err, domain.ErrTokenUnknown) {
		t.Errorf("compte fermé, jeton déjà émis : attendu ErrTokenUnknown, obtenu %v", err)
	}
	if err := mod.Authorize(ctx, id, cancel); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("compte fermé, autorisation : attendu ErrForbidden, obtenu %v", err)
	}
}

// TestDeactivationIsIdempotentAndReversible garde les deux sens du geste.
//
// Idempotente : deux administrateurs qui réagissent au même incident ne doivent
// pas s'annuler. Réversible : la fermeture est parfois une erreur, et un module
// qui ne saurait que fermer ferait réparer cela à la main dans le magasin — donc
// sans trace, et par quelqu'un qui a désormais un accès direct aux comptes.
func TestDeactivationIsIdempotentAndReversible(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	for range 2 {
		if err := mod.Deactivate(ctx, id); err != nil {
			t.Fatalf("fermeture répétée: %v", err)
		}
	}

	if err := mod.Reactivate(ctx, id); err != nil {
		t.Fatalf("réouverture: %v", err)
	}
	if _, err := mod.Authenticate(ctx, subject, secret); err != nil {
		t.Fatalf("le compte rouvert doit s'authentifier : %v", err)
	}
}

// TestDeactivatingAnUnknownIdentityIsRefused interdit de fermer un compte
// imaginaire en silence.
//
// Un succès sur un identifiant inconnu ferait croire à l'administrateur qu'il
// vient de fermer le compte compromis — alors qu'il s'est trompé de ligne, et que
// le vrai compte est toujours ouvert.
func TestDeactivatingAnUnknownIdentityIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	if err := mod.Deactivate(ctx, "personne"); err == nil {
		t.Error("fermer une identité inconnue doit être refusé")
	}
	if err := mod.Deactivate(ctx, ""); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("identité vide : attendu ErrIncomplete, obtenu %v", err)
	}
}
