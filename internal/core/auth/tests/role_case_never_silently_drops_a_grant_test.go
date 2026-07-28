package tests

import (
	"context"
	"testing"
)

// TestRoleCaseNeverSilentlyDropsAGrant : `Comptable` et `comptable` sont UN rôle.
//
// # Pourquoi ce test existe (ADR 013)
//
// `NewRole` normalise le nom du rôle et `NewIdentity` normalise ceux qu'elle
// reçoit — mais l'affectation passe par un chemin distinct. Si ce chemin ne
// normalise pas, le rôle est retenu sous `Comptable`, cherché sous `comptable`,
// et n'accorde RIEN.
//
// La faute est de la pire catégorie : elle ne produit aucune erreur. Un
// administrateur voit le rôle affecté dans l'interface, la personne concernée
// reçoit un 403, et les deux ont raison de ne pas comprendre. Rien dans les
// journaux ne mentionne une casse.
func TestRoleCaseNeverSilentlyDropsAGrant(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	if err := mod.DefineRole(ctx, "Comptable", []string{"billing.invoice.cancel"}); err != nil {
		t.Fatalf("définition du rôle: %v", err)
	}
	if err := mod.AssignRoles(ctx, id, []string{"  COMPTABLE  "}); err != nil {
		t.Fatalf("affectation du rôle: %v", err)
	}

	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.cancel")); err != nil {
		t.Fatalf("la casse du rôle ne doit jamais faire perdre une permission : %v", err)
	}
}

// TestAssigningAnUndefinedRoleGrantsNothingAndFailsNot sépare l'ordre de
// provisionnement de la décision de sécurité.
//
// Affecter un rôle qui n'existe pas encore n'accorde RIEN — `Grants` ne trouve
// aucune permission — mais reste permis, pour qu'on puisse provisionner dans
// l'ordre qu'on veut. Refuser serait une contrainte d'ordonnancement déguisée en
// règle de sécurité, et elle ferait échouer un import de données parfaitement
// valide dont les lignes ne sont pas triées.
//
// La propriété qui compte est la seconde : l'affectation ne doit rien accorder
// tant que le rôle n'est pas défini.
func TestAssigningAnUndefinedRoleGrantsNothingAndFailsNot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)
	id := register(t, mod, subject)

	if err := mod.AssignRoles(ctx, id, []string{"jamais-defini"}); err != nil {
		t.Fatalf("l'affectation ne doit pas dépendre de l'ordre de provisionnement : %v", err)
	}
	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.cancel")); err == nil {
		t.Fatal("un rôle non défini ne doit accorder aucune permission")
	}

	// Une fois défini, il accorde — sans réaffectation.
	if err := mod.DefineRole(ctx, "jamais-defini", []string{"billing.invoice.cancel"}); err != nil {
		t.Fatalf("définition tardive du rôle: %v", err)
	}
	if err := mod.Authorize(ctx, id, permission(t, "billing.invoice.cancel")); err != nil {
		t.Fatalf("le rôle défini après coup doit accorder : %v", err)
	}
}

// TestAssigningRolesToAnUnknownIdentityIsRefused interdit le succès silencieux.
//
// Un succès silencieux ferait croire à l'administrateur que le droit est posé.
// C'est la même faute que fermer un compte inexistant sans le dire, et elle se
// découvre au même moment : quand quelqu'un ne peut pas faire ce qu'on lui a
// pourtant accordé.
func TestAssigningRolesToAnUnknownIdentityIsRefused(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mod, _ := newModule(t, nil)

	if err := mod.AssignRoles(ctx, "personne", []string{"comptable"}); err == nil {
		t.Error("affecter un rôle à une identité inconnue doit être refusé")
	}
	if err := mod.AssignRoles(ctx, "", []string{"comptable"}); err == nil {
		t.Error("affecter un rôle à une identité vide doit être refusé")
	}
}
