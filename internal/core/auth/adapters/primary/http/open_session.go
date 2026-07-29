package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// openSessionInput is the sign-in request.
//
// # No validation constraint in the schema, and that is deliberate
//
// A length bound here would make the request be refused BEFORE the domain, with
// the library's message rather than the module's — and above all, it would
// distinguish "secret too short" from "invalid credentials". That would reopen
// the account existence oracle through schema validation, after having closed
// it in the use case.
//
// The schema DOCUMENTS, it does not validate.
type openSessionInput struct {
	Body struct {
		Subject string `json:"subject" doc:"What designates the account — address, identifier" example:"alice@example.com"`
		Secret  string `json:"secret"  doc:"Plain secret. Never logged, never returned."`
	}
}

// openSessionOutput carries the opened session.
//
// The body is the type of the PUBLISHED LANGUAGE, not the domain's. Serialising
// a `domain.Session` would expose whatever the domain added to it tomorrow;
// here, what goes out is enumerated.
type openSessionOutput struct {
	Status int
	Body   contract.SessionResponse
}

// mountOpenSession exposes the exchange of a secret for a token.
func mountOpenSession(api huma.API, mod auth.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "open-session",
		Method:      contract.OpenSessionRoute.Method,
		Path:        contract.OpenSessionRoute.Path,
		Summary:     "Open a session",
		Description: "Exchanges a secret for an opaque token. " +
			"The token AUTHENTICATES: it carries no permission, and the rights " +
			"are re-read on every decision (ADR 017).",
		Tags:          []string{apiTag},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *openSessionInput) (*openSessionOutput, error) {
		session, err := mod.Authenticate(ctx, in.Body.Subject, in.Body.Secret)
		if err != nil {
			return nil, statusFor(err)
		}

		return &openSessionOutput{
			Status: http.StatusCreated,
			Body: contract.SessionResponse{
				Token:      session.Token.String(),
				IdentityID: string(session.Identity),
				ExpiresAt:  session.ExpiresAt,
			},
		}, nil
	})
}
