package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestPermissionShapeIsEnforced : `domaine.ressource.action`, en minuscules,
// trois segments exactement.
//
// # Ce que la contrainte évite
//
// Une permission est une chaîne : rien n'empêche d'écrire `admin` d'un côté et
// `Admin` de l'autre. Les deux coexisteraient, l'une accorderait, l'autre pas, et
// le défaut se manifesterait comme un « il a le droit mais ça refuse » —
// impossible à diagnostiquer sans comparer deux chaînes à l'œil.
//
// Le refus est à la CONSTRUCTION : le champ est privé, donc une permission
// invalide ne peut pas exister hors du domaine.
func TestPermissionShapeIsEnforced(t *testing.T) {
	t.Parallel()

	refused := []string{
		"",
		"admin",                      // un segment
		"billing.invoice",            // deux
		"billing.invoice.cancel.now", // quatre
		"Billing.Invoice.Cancel",     // capitales — normalisées, donc acceptées
		"billing..cancel",            // segment vide
		"1billing.invoice.cancel",    // commence par un chiffre
		"billing.invoice.can-cel",    // tiret
		"billing invoice cancel",     // espaces
	}
	for _, raw := range refused {
		_, err := domain.NewPermission(raw)
		if raw == "Billing.Invoice.Cancel" {
			if err != nil {
				t.Errorf("la casse doit être NORMALISÉE, pas refusée : %q → %v", raw, err)
			}
			continue
		}
		if !errors.Is(err, domain.ErrIncomplete) {
			t.Errorf("permission %q : attendu ErrIncomplete, obtenu %v", raw, err)
		}
	}

	accepted, err := domain.NewPermission("  BILLING.Invoice.Cancel  ")
	if err != nil {
		t.Fatalf("permission valide refusée: %v", err)
	}
	if accepted.String() != "billing.invoice.cancel" {
		t.Fatalf("normalisation attendue `billing.invoice.cancel`, obtenu %q", accepted.String())
	}
}

// TestDefiningARoleRefusesTheWholeSetOnOneBadPermission garde l'atomicité de la
// définition.
//
// Un rôle à moitié défini serait pire qu'un rôle refusé : il accorderait quelque
// chose, sans que personne sache quoi. Le test constate aussi que la définition
// fautive ne laisse RIEN derrière elle.
func TestDefiningARoleRefusesTheWholeSetOnOneBadPermission(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	err := mod.DefineRole(ctx, "comptable", []string{
		"billing.invoice.cancel",
		"pas une permission",
		"billing.invoice.read",
	})
	if !errors.Is(err, domain.ErrIncomplete) {
		t.Fatalf("attendu ErrIncomplete, obtenu %v", err)
	}

	if err := mod.AssignRoles(ctx, id, []string{"comptable"}); err != nil {
		t.Fatalf("affectation: %v", err)
	}
	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.cancel")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("la définition a échoué : aucune permission ne doit avoir été accordée, obtenu %v", err)
	}
}

// TestRoleGrantsWithoutWildcard : la comparaison est exacte, sans joker.
//
// Un `billing.*` ferait gagner quelques lignes de configuration et rendrait
// impossible de répondre à « qui peut annuler une facture ? » autrement qu'en
// exécutant le code. La question se pose en audit, et elle se pose sur des
// permissions qu'on n'a pas écrites soi-même.
func TestRoleGrantsWithoutWildcard(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)
	grant(t, mod, id, "lecteur", "billing.invoice.read")

	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.read")); err != nil {
		t.Fatalf("la permission accordée doit valoir : %v", err)
	}

	for _, other := range []string{"billing.invoice.cancel", "billing.report.read", "admin.user.read"} {
		if err := mod.Authorize(ctx, id, permission(t, other)); !errors.Is(err, domain.ErrForbidden) {
			t.Errorf("%q n'est pas accordée : attendu ErrForbidden, obtenu %v", other, err)
		}
	}
}

// TestRoleAbsorbsDuplicatesAndSortsPermissions fixe la forme canonique d'un rôle.
//
// Les doublons sont ABSORBÉS plutôt que refusés : deux fois la même permission
// exprime la même intention, et refuser ferait échouer un import de données pour
// une redondance sans conséquence.
//
// Le tri rend l'égalité de deux rôles observable et les messages comparables d'une
// exécution à l'autre — un ordre de map change à chaque parcours.
func TestRoleAbsorbsDuplicatesAndSortsPermissions(t *testing.T) {
	t.Parallel()

	read := permission(t, "billing.invoice.read")
	cancel := permission(t, "billing.invoice.cancel")

	role, err := domain.NewRole("  Comptable  ", []domain.Permission{read, cancel, read})
	if err != nil {
		t.Fatalf("rôle: %v", err)
	}
	if role.Name != "comptable" {
		t.Fatalf("le nom du rôle doit être normalisé, obtenu %q", role.Name)
	}
	if len(role.Permissions) != 2 {
		t.Fatalf("les doublons doivent être absorbés, obtenu %v", role.Permissions)
	}
	if role.Permissions[0].String() != "billing.invoice.cancel" {
		t.Fatalf("les permissions doivent être triées, obtenu %v", role.Permissions)
	}

	if _, err := domain.NewRole("   ", nil); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("un rôle sans nom : attendu ErrIncomplete, obtenu %v", err)
	}
	if _, err := domain.NewRole("comptable", []domain.Permission{{}}); !errors.Is(err, domain.ErrIncomplete) {
		t.Errorf("une permission non construite : attendu ErrIncomplete, obtenu %v", err)
	}
}
