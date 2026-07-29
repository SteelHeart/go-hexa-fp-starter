package http

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// identityOutput carries the resolved identity.
type identityOutput struct {
	Body contract.IdentityResponse
}

// mountIdentity exposes the resolution of the token into an identity.
//
// # What this route does NOT do
//
// It authorises nothing. It says WHO presents the token, and a client that
// deduced from it what they are allowed to do would be wrong at the first
// withdrawal of a right. The decision must be asked for every time — that is
// decision 1 of ADR 017, and it is also why the response carries no
// permission.
func mountIdentity(api huma.API, mod auth.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "resolve-identity",
		Method:      contract.IdentityRoute.Method,
		Path:        contract.IdentityRoute.Path,
		Summary:     "Resolve the presented token",
		Description: "Returns the identity attached to the token. " +
			"Authorises NOTHING: the token authenticates, it does not authorise.",
		Tags: []string{apiTag},
	}, func(ctx context.Context, in *bearerInput) (*identityOutput, error) {
		token, err := bearer(in.Authorization)
		if err != nil {
			return nil, err
		}

		identity, err := mod.Verify(ctx, token)
		if err != nil {
			return nil, statusFor(err)
		}

		return &identityOutput{Body: contract.IdentityResponse{
			IdentityID: string(identity.ID),
			Subject:    identity.Subject.String(),
			Roles:      identity.Roles,
			CreatedAt:  identity.CreatedAt,
		}}, nil
	})
}
