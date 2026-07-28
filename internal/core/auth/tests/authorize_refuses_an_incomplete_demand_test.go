package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// TestAuthorizeRefusesAnIncompleteDemand : la valeur zéro n'ouvre rien.
//
// # Le cas qu'un magasin permissif rendrait catastrophique
//
// Une `Permission{}` non construite porte la chaîne vide, et une `IdentityID`
// vide aussi. Sans ce refus en amont, les deux atteindraient le magasin où elles
// deviendraient des clés légitimes — et la première ligne enregistrée sous une
// clé vide accorderait à quiconque n'envoie rien.
//
// Le refus est distinct de `ErrForbidden` : la demande elle-même est mal formée,
// ce qu'une surface HTTP traduit en 422 et non en 403.
func TestAuthorizeRefusesAnIncompleteDemand(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)
	grant(t, mod, id, "comptable", "billing.invoice.cancel")

	cases := map[string]struct {
		identity   domain.IdentityID
		permission domain.Permission
	}{
		"identité vide":             {"", permission(t, "billing.invoice.cancel")},
		"permission non construite": {id, domain.Permission{}},
		"les deux":                  {"", domain.Permission{}},
	}

	for name, tc := range cases {
		err := mod.Authorize(ctx, tc.identity, tc.permission)
		if !errors.Is(err, domain.ErrIncomplete) {
			t.Errorf("%s : attendu ErrIncomplete, obtenu %v", name, err)
		}
		if errors.Is(err, domain.ErrForbidden) {
			t.Errorf("%s : une demande mal formée n'est pas un refus de permission", name)
		}
	}
}

// TestAuthorizeRefusesAnUnknownIdentity : un identifiant inventé n'accorde rien.
//
// Deny par défaut jusqu'au bout : `Grants` rend `false` pour ce qu'il ne connaît
// pas, et le cas d'usage traduit en `ErrForbidden`. La faute inverse — traiter
// « inconnu » comme « pas de restriction connue » — est la manière dont un
// contrôle d'accès s'ouvre entièrement en une ligne.
func TestAuthorizeRefusesAnUnknownIdentity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	err := mod.Authorize(ctx, "identifiant-invente", permission(t, "billing.invoice.cancel"))
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("identité inconnue : attendu ErrForbidden, obtenu %v", err)
	}
}

// TestVerifyAndAuthorizeStayTwoCalls constate la SÉPARATION des deux gestes.
//
// `Verify` rend une identité et ne dit rien des permissions ; `Authorize` rend un
// refus ou rien et ne dit rien de l'identité. Les fusionner en un
// `verifyAndAuthorize(token, permission)` ramènerait les permissions dans le
// périmètre du jeton par la porte de derrière, et la prochaine optimisation
// naturelle serait de les y mettre pour de bon.
//
// Le test constate qu'une identité PARFAITEMENT valide n'obtient rien sans
// permission accordée.
func TestVerifyAndAuthorizeStayTwoCalls(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	session, err := mod.Authenticate(ctx, subject, secret)
	if err != nil {
		t.Fatalf("authentification: %v", err)
	}

	identity, err := mod.Verify(ctx, session.Token)
	if err != nil {
		t.Fatalf("le jeton est valide : %v", err)
	}
	if identity.ID != id {
		t.Fatalf("attendu l'identité %q, obtenu %q", id, identity.ID)
	}

	if err := mod.Authorize(ctx, identity.ID, permission(t, "billing.invoice.cancel")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("authentifié n'est pas autorisé : attendu ErrForbidden, obtenu %v", err)
	}
}
