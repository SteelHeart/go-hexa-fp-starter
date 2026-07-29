package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// createIdentityInput creates an account. The bearer must be authorised.
type createIdentityInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token, in the form: Bearer <token>"`
	Body          contract.CreateIdentityRequest
}

// createIdentityOutput carries the created identity.
type createIdentityOutput struct {
	Status int
	Body   contract.IdentityResponse
}

// roleInput defines a role. The bearer must be authorised.
type roleInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token, in the form: Bearer <token>"`
	Name          string `path:"name" doc:"Role name"`
	Body          contract.DefineRoleRequest
}

// identityRolesInput assigns roles. The bearer must be authorised.
type identityRolesInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token, in the form: Bearer <token>"`
	ID            string `path:"id" doc:"Identifier of the identity"`
	Body          contract.AssignRolesRequest
}

// identityInput designates an identity. The bearer must be authorised.
type identityInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token, in the form: Bearer <token>"`
	ID            string `path:"id" doc:"Identifier of the identity"`
}

// emptyOutput carries nothing: 204.
type emptyOutput struct {
	Status int
}

// mountAdministration exposes the ADMINISTRATION operations, all protected.
//
// # What these routes close
//
// Issue #99 had declared them out of scope, for want of a first administrator:
// exposing them without protecting them would have opened account creation and
// the granting of rights to anyone. With bootstrapping resolved, they arrive —
// and each one starts with a line that NAMES its permission.
//
// # Every operation requires a DISTINCT permission
//
// A single `auth.admin` permission would be simpler and would give whoever can
// create an account the right to close the others. The granularity is what lets
// a "support" role reopen an account without being able to grant themselves
// rights.
func mountAdministration(api huma.API, mod auth.Module) {
	guard := Guard{Module: mod}
	mountCreateIdentity(api, mod, guard)
	mountDefineRole(api, mod, guard)
	mountAssignRoles(api, mod, guard)
	mountCloseIdentity(api, mod, guard)
}

// mountCreateIdentity exposes the creation of an account.
func mountCreateIdentity(api huma.API, mod auth.Module, guard Guard) {
	permission := mustPermission(contract.PermissionIdentityCreate)
	huma.Register(api, huma.Operation{
		OperationID:   "create-identity",
		Method:        contract.CreateIdentityRoute.Method,
		Path:          contract.CreateIdentityRoute.Path,
		Summary:       "Create an account",
		Description:   "Requires the `" + contract.PermissionIdentityCreate + "` permission.",
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

// mountDefineRole exposes the definition of a role.
func mountDefineRole(api huma.API, mod auth.Module, guard Guard) {
	permission := mustPermission(contract.PermissionRoleWrite)
	huma.Register(api, huma.Operation{
		OperationID: "define-role",
		Method:      contract.DefineRoleRoute.Method,
		Path:        contract.DefineRoleRoute.Path,
		Summary:     "Define a role",
		Description: "REPLACES the role and its permissions. Requires `" +
			contract.PermissionRoleWrite + "` — the most powerful permission of the module: " +
			"whoever holds it can grant themselves all the others.",
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

// mountAssignRoles exposes the assignment of roles.
func mountAssignRoles(api huma.API, mod auth.Module, guard Guard) {
	permission := mustPermission(contract.PermissionIdentityRoles)
	huma.Register(api, huma.Operation{
		OperationID:   "assign-roles",
		Method:        contract.AssignRolesRoute.Method,
		Path:          contract.AssignRolesRoute.Path,
		Summary:       "Assign roles",
		Description:   "REPLACES the roles. Requires `" + contract.PermissionIdentityRoles + "`.",
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

// mountCloseIdentity exposes the closure of an account.
//
// This is the gesture you make when you discover an account is compromised,
// hence the only moment where latency really matters. It takes effect on the
// next call, without any token expiry — decision 1 of ADR 017.
func mountCloseIdentity(api huma.API, mod auth.Module, guard Guard) {
	permission := mustPermission(contract.PermissionIdentityClose)
	huma.Register(api, huma.Operation{
		OperationID: "close-identity",
		Method:      contract.CloseIdentityRoute.Method,
		Path:        contract.CloseIdentityRoute.Path,
		Summary:     "Close an account",
		Description: "Takes effect IMMEDIATELY: tokens already issued stop being worth anything. " +
			"Requires `" + contract.PermissionIdentityClose + "`.",
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
