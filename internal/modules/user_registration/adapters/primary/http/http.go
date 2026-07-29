// Package http exposes the registration module on the HTTP surface.
//
// # A surface is a TRANSLATOR, never a place for business
//
// This package does exactly three things: translate an HTTP request into a
// domain command, call the use case, translate the Result into an HTTP response.
// It validates nothing itself — the domain already does, and duplicating the
// validation guarantees that one day the two will diverge.
//
// Adding a surface (CLI, gRPC, event consumer) amounts to writing a sibling of
// this file. Not a line of `domain/` or `application/` moves: that is the
// property ADR 005 buys.
package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	contract "github.com/SteelHeart/go-hexa-fp-starter/internal/contracts/userregistration"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// registerInput is the registration request.
//
// # No validation constraint in the schema, and that is deliberate
//
// The first version carried `format:"email"`. The outcome, observed while
// running the server: huma refused the address BEFORE the domain, and returned
// its own message — "expected string to be RFC 5322 email" — in English, in the
// middle of an API whose every other message comes from the domain and is in
// French.
//
// Two validation rules for the same field is one too many: they will diverge,
// and it is the wrong one that will speak to the user. The schema therefore
// DOCUMENTS (doc, example), it does not validate. `domain.NewEmail` remains the
// sole judge, and it is its message that travels up.
type registerInput struct {
	Body struct {
		Email    string `json:"email"    doc:"Email address" example:"alice@example.com"`
		Password string `json:"password" doc:"Plaintext password. Never logged, never returned."`
	}
}

// registerOutput carries the registration response.
//
// The body is the type of the PUBLISHED LANGUAGE, not the domain type.
// Serialising `domain.User` would expose the password digest the day someone
// adds a JSON tag — here, that is structurally impossible.
type registerOutput struct {
	Status int
	Body   contract.RegisterResponse
}

// Mount registers the module's operations on the API.
//
// Receives the Module, never a driver nor a pool: a surface cannot bypass the
// use case, not even by accident.
func Mount(api huma.API, mod userregistration.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "register-user",
		Method:      contract.RegisterRoute.Method,
		Path:        contract.RegisterRoute.Path,
		Summary:     "Register a user",
		Description: "Creates an account awaiting confirmation. " +
			"The account is born `pending`: the address is not proven yet.",
		Tags:          []string{"users"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *registerInput) (*registerOutput, error) {
		command := domain.RegistrationCommand{
			Email:    in.Body.Email,
			Password: in.Body.Password,
		}

		value, failure, ok := mod.Register(ctx, command).Get()
		if !ok {
			return nil, statusFor(failure)
		}

		return &registerOutput{
			Status: http.StatusCreated,
			Body: contract.RegisterResponse{
				UserID:    value.ID.String(),
				Email:     value.Email.String(),
				Status:    string(value.Status),
				CreatedAt: value.CreatedAt,
			},
		}, nil
	})
}

// statusFor translates a domain error into an HTTP response.
//
// The `switch` is EXHAUSTIVE, and the `exhaustive` linter fails the CI if it
// stops being so. That is the intended effect: adding an error code to the
// domain forces a decision, here, about what the outside world sees of it.
// Without that constraint, a new code would silently translate into a 500 — and
// a user input mistake would be counted as a service outage.
//
// The `default` stays, despite exhaustiveness: it covers the zero value and any
// string fabricated by a conversion. Deny by default — the unknown is treated as
// a breakdown, never as an acceptable request.
func statusFor(err domain.Error) error {
	switch err.Code {
	case domain.CodeInvalidEmail, domain.CodeWeakPassword:
		return huma.Error422UnprocessableEntity(err.Message, fieldDetail(err))

	case domain.CodeEmailAlreadyExists:
		// 409 and not 422: the request is well formed, it is the STATE of the
		// server that opposes it. The distinction matters to a client deciding
		// whether or not to retry.
		return huma.Error409Conflict(err.Message, fieldDetail(err))

	case domain.CodeUnavailable:
		// 503 and not 500: transient. That is what authorises a client to
		// retry, and what avoids waking someone up for a database that is
		// restarting.
		return huma.Error503ServiceUnavailable(err.Message)

	case domain.CodeInternal:
		// The technical cause is NEVER returned to the caller: an SQL error
		// sent back to the client is a structure leak. It is logged by the
		// middleware, which ties it to the correlation identifier.
		return huma.Error500InternalServerError("une erreur interne est survenue")

	default:
		return huma.Error500InternalServerError("une erreur interne est survenue")
	}
}

// fieldDetail names the faulty field when the domain stated it.
//
// Without it, a form would not know which field to highlight and would display
// the error at the top of the page — which, on a two-field form, forces the user
// to guess.
func fieldDetail(err domain.Error) error {
	if err.Field == "" {
		return nil
	}
	return &huma.ErrorDetail{
		Location: "body." + err.Field,
		Message:  err.Message,
	}
}

// availabilityOutput carries the availability response.
type availabilityOutput struct {
	Body struct {
		Available bool `json:"available" doc:"True if the address is still free"`
	}
}

type availabilityInput struct {
	Email string `query:"email" required:"true" doc:"Address to check"`
}

// MountAvailability exposes the availability check for an email address.
//
// Separate from Mount: this is a READ port, and it has neither the same caching
// needs nor the same rate-limiting needs as a write. Mounting them together
// would suggest they are governed alike.
//
// ⚠️ Accepted NON-GUARANTEE: this endpoint allows ENUMERATING the registered
// addresses. That is acceptable as long as it is rate limited; it is no longer
// acceptable without. The `ratelimit` module does not exist yet — see
// SECURITY.md.
func MountAvailability(api huma.API, mod userregistration.Module) {
	huma.Register(api, huma.Operation{
		OperationID: "check-email-availability",
		Method:      http.MethodGet,
		Path:        "/v1/users/availability",
		Summary:     "Check the availability of an address",
		Tags:        []string{"users"},
	}, func(ctx context.Context, in *availabilityInput) (*availabilityOutput, error) {
		available, failure, ok := mod.CheckEmail(ctx, in.Email).Get()
		if !ok {
			return nil, statusFor(failure)
		}

		out := &availabilityOutput{}
		out.Body.Available = available
		return out, nil
	})
}
