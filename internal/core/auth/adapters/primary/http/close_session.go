package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// bearerInput carries the presented token.
//
// The token goes through the `Authorization` header and not through the body or
// the URL: a URL ends up in access logs, in the browser history and in the
// `Referer` header sent to the first third-party site visited afterwards. A
// token in a URL is a published token.
type bearerInput struct {
	Authorization string `header:"Authorization" doc:"Bearer token, in the form: Bearer <token>"`
}

// closeSessionOutput carries nothing.
//
// 204 and not 200 with a "signed out" body: there is nothing to say, and an
// invented body would give a client the idea of reading it.
type closeSessionOutput struct {
	Status int
}

// mountCloseSession exposes the revocation of the presented token.
func mountCloseSession(api huma.API, mod auth.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "close-session",
		Method:      contract.CloseSessionRoute.Method,
		Path:        contract.CloseSessionRoute.Path,
		Summary:     "Close the current session",
		Description: "Revokes the presented token, IMMEDIATELY. " +
			"Idempotent: closing an already closed session returns 204.",
		Tags:          []string{apiTag},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *bearerInput) (*closeSessionOutput, error) {
		token, err := bearer(in.Authorization)
		if err != nil {
			return nil, err
		}
		if err := mod.Revoke(ctx, token); err != nil {
			return nil, statusFor(err)
		}
		return &closeSessionOutput{Status: http.StatusNoContent}, nil
	})
}
