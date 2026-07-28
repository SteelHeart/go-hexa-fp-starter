package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestRevokedPermissionIsRefusedOnTheNextCall est LE témoin de l'ADR 017.
//
// # Ce qu'il constate
//
// Une permission retirée cesse d'accorder à l'appel SUIVANT, sans qu'aucun jeton
// n'expire et sans que personne ne se reconnecte. Le jeton reste valide tout du
// long — le test le vérifie explicitement entre les deux autorisations, sinon un
// refus pourrait venir d'une session invalidée et la démonstration serait vide.
//
// # Pourquoi ce test existe (ADR 013)
//
// Il échouerait le jour où quelqu'un déplacerait les permissions dans le jeton,
// ou ajouterait un cache non borné devant `Grants`. Les deux sont des
// optimisations tentantes, et les deux rouvrent la même fenêtre : un accès
// révoqué qui fonctionne encore. Or le jour où l'on révoque, c'est qu'on est
// pressé.
func TestRevokedPermissionIsRefusedOnTheNextCall(t *testing.T) {
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
	if err := mod.Authorize(ctx, id, cancel); err != nil {
		t.Fatalf("la permission vient d'être accordée, elle doit valoir : %v", err)
	}

	// Révocation : le rôle est REDÉFINI sans la permission. Aucun jeton n'est
	// touché, aucune session n'est supprimée.
	if err := mod.DefineRole(ctx, "comptable", nil); err != nil {
		t.Fatalf("révocation de la permission: %v", err)
	}

	// Le jeton vaut TOUJOURS : c'est ce qui rend le refus qui suit concluant.
	if _, err := mod.Verify(ctx, session.Token); err != nil {
		t.Fatalf("le jeton ne devait pas être affecté par la révocation : %v", err)
	}

	if err := mod.Authorize(ctx, id, cancel); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("permission révoquée : attendu ErrForbidden, obtenu %v", err)
	}
}

// TestRevokedRoleIsRefusedOnTheNextCall constate la même chose par l'autre bout.
//
// Retirer le RÔLE plutôt que la permission doit produire le même refus immédiat.
// Les deux chemins existent — on retire un droit à tout le monde, ou on retire
// quelqu'un d'un groupe — et un seul des deux couvert laisserait l'autre dériver.
func TestRevokedRoleIsRefusedOnTheNextCall(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	id := register(t, mod, subject)
	cancel := permission(t, "billing.invoice.cancel")
	grant(t, mod, id, "comptable", "billing.invoice.cancel")

	if err := mod.Authorize(ctx, id, cancel); err != nil {
		t.Fatalf("la permission vient d'être accordée, elle doit valoir : %v", err)
	}

	if err := mod.AssignRoles(ctx, id, nil); err != nil {
		t.Fatalf("retrait des rôles: %v", err)
	}

	if err := mod.Authorize(ctx, id, cancel); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("rôle retiré : attendu ErrForbidden, obtenu %v", err)
	}
}
