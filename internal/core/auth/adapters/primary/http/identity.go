package http

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// identityOutput porte l'identité résolue.
type identityOutput struct {
	Body contract.IdentityResponse
}

// mountIdentity expose la résolution du jeton en identité.
//
// # Ce que cette route ne fait PAS
//
// Elle n'autorise rien. Elle dit QUI présente le jeton, et un client qui en
// déduirait ce qu'il a le droit de faire se tromperait au premier retrait de
// droit. La décision se demande à chaque fois — c'est la décision 1 de
// l'ADR 017, et c'est aussi pourquoi la réponse ne porte aucune permission.
func mountIdentity(api huma.API, mod auth.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "resolve-identity",
		Method:      contract.IdentityRoute.Method,
		Path:        contract.IdentityRoute.Path,
		Summary:     "Résoudre le jeton présenté",
		Description: "Rend l'identité rattachée au jeton. " +
			"N'autorise RIEN : le jeton authentifie, il n'autorise pas.",
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
