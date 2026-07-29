package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// createIdentityInput crée un compte. Le porteur doit être autorisé.
type createIdentityInput struct {
	Authorization string `header:"Authorization" doc:"Jeton porteur, sous la forme: Bearer <jeton>"`
	Body          contract.CreateIdentityRequest
}

// createIdentityOutput porte l'identité créée.
type createIdentityOutput struct {
	Status int
	Body   contract.IdentityResponse
}

// roleInput définit un rôle. Le porteur doit être autorisé.
type roleInput struct {
	Authorization string `header:"Authorization" doc:"Jeton porteur, sous la forme: Bearer <jeton>"`
	Name          string `path:"name" doc:"Nom du rôle"`
	Body          contract.DefineRoleRequest
}

// identityRolesInput affecte des rôles. Le porteur doit être autorisé.
type identityRolesInput struct {
	Authorization string `header:"Authorization" doc:"Jeton porteur, sous la forme: Bearer <jeton>"`
	ID            string `path:"id" doc:"Identifiant de l'identité"`
	Body          contract.AssignRolesRequest
}

// identityInput désigne une identité. Le porteur doit être autorisé.
type identityInput struct {
	Authorization string `header:"Authorization" doc:"Jeton porteur, sous la forme: Bearer <jeton>"`
	ID            string `path:"id" doc:"Identifiant de l'identité"`
}

// emptyOutput ne porte rien : 204.
type emptyOutput struct {
	Status int
}

// mountAdministration expose les opérations d'ADMINISTRATION, toutes protégées.
//
// # Ce que ces routes ferment
//
// L'issue #99 les avait déclarées hors périmètre, faute d'un premier
// administrateur : les exposer sans les protéger aurait ouvert la création de
// comptes et l'attribution de droits à quiconque. L'amorçage résolu, elles
// arrivent — et chacune commence par une ligne qui NOMME sa permission.
//
// # Chaque opération exige une permission DISTINCTE
//
// Une seule permission `auth.admin` serait plus simple et donnerait à qui sait
// créer un compte le droit de fermer les autres. La granularité est ce qui
// permet à un rôle « support » de rouvrir un compte sans pouvoir s'accorder de
// droits.
func mountAdministration(api huma.API, mod auth.Module) {
	guard := Guard{Module: mod}
	mountCreateIdentity(api, mod, guard)
	mountDefineRole(api, mod, guard)
	mountAssignRoles(api, mod, guard)
	mountCloseIdentity(api, mod, guard)
}

// mountCreateIdentity expose la création d'un compte.
func mountCreateIdentity(api huma.API, mod auth.Module, guard Guard) {
	permission := mustPermission(contract.PermissionIdentityCreate)
	huma.Register(api, huma.Operation{
		OperationID:   "create-identity",
		Method:        contract.CreateIdentityRoute.Method,
		Path:          contract.CreateIdentityRoute.Path,
		Summary:       "Créer un compte",
		Description:   "Exige la permission `" + contract.PermissionIdentityCreate + "`.",
		Tags:          []string{apiTag},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createIdentityInput) (*createIdentityOutput, error) {
		if _, err := guard.Require(ctx, in.Authorization, permission); err != nil {
			return nil, err
		}

		identity, err := mod.Register(ctx, in.Body.Subject, in.Body.Secret)
		if err != nil {
			return nil, statusFor(err)
		}
		return &createIdentityOutput{Status: http.StatusCreated, Body: contract.IdentityResponse{
			IdentityID: string(identity.ID),
			Subject:    identity.Subject.String(),
			Roles:      identity.Roles,
			CreatedAt:  identity.CreatedAt,
		}}, nil
	})
}

// mountDefineRole expose la définition d'un rôle.
func mountDefineRole(api huma.API, mod auth.Module, guard Guard) {
	permission := mustPermission(contract.PermissionRoleWrite)
	huma.Register(api, huma.Operation{
		OperationID: "define-role",
		Method:      contract.DefineRoleRoute.Method,
		Path:        contract.DefineRoleRoute.Path,
		Summary:     "Définir un rôle",
		Description: "REMPLACE le rôle et ses permissions. Exige `" +
			contract.PermissionRoleWrite + "` — la permission la plus puissante du module : " +
			"qui la détient peut s'accorder toutes les autres.",
		Tags:          []string{apiTag},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *roleInput) (*emptyOutput, error) {
		if _, err := guard.Require(ctx, in.Authorization, permission); err != nil {
			return nil, err
		}
		if err := mod.DefineRole(ctx, in.Name, in.Body.Permissions); err != nil {
			return nil, statusFor(err)
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})
}

// mountAssignRoles expose l'affectation de rôles.
func mountAssignRoles(api huma.API, mod auth.Module, guard Guard) {
	permission := mustPermission(contract.PermissionIdentityRoles)
	huma.Register(api, huma.Operation{
		OperationID:   "assign-roles",
		Method:        contract.AssignRolesRoute.Method,
		Path:          contract.AssignRolesRoute.Path,
		Summary:       "Affecter des rôles",
		Description:   "REMPLACE les rôles. Exige `" + contract.PermissionIdentityRoles + "`.",
		Tags:          []string{apiTag},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *identityRolesInput) (*emptyOutput, error) {
		if _, err := guard.Require(ctx, in.Authorization, permission); err != nil {
			return nil, err
		}
		if err := mod.AssignRoles(ctx, domain.IdentityID(in.ID), in.Body.Roles); err != nil {
			return nil, statusFor(err)
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})
}

// mountCloseIdentity expose la fermeture d'un compte.
//
// C'est le geste qu'on fait quand on découvre qu'un compte est compromis, donc
// le seul moment où la latence compte vraiment. Il prend effet au prochain appel,
// sans expiration de jeton — décision 1 de l'ADR 017.
func mountCloseIdentity(api huma.API, mod auth.Module, guard Guard) {
	permission := mustPermission(contract.PermissionIdentityClose)
	huma.Register(api, huma.Operation{
		OperationID: "close-identity",
		Method:      contract.CloseIdentityRoute.Method,
		Path:        contract.CloseIdentityRoute.Path,
		Summary:     "Fermer un compte",
		Description: "Prend effet IMMÉDIATEMENT : les jetons déjà émis cessent de valoir. " +
			"Exige `" + contract.PermissionIdentityClose + "`.",
		Tags:          []string{apiTag},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *identityInput) (*emptyOutput, error) {
		if _, err := guard.Require(ctx, in.Authorization, permission); err != nil {
			return nil, err
		}
		if err := mod.Deactivate(ctx, domain.IdentityID(in.ID)); err != nil {
			return nil, statusFor(err)
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})
}
