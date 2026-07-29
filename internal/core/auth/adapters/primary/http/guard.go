package http

import (
	"context"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// Guard protects an operation with a permission.
//
// # What this type finally makes reachable
//
// `Authorize` was proven by the module's tests and exposed by no surface:
// nobody could
// use it from the outside. It was not written earlier deliberately — a guard
// with no route to protect is dead code, and this repository has just closed
// three of those. It arrives with its first routes.
//
// # Why an EXPLICIT call and not a middleware
//
// A middleware that dropped the identity into the context would violate
// decision 5 of ADR 017 — *the identity enters through the command, never
// through the context*. It would also make it INVISIBLE that an operation
// depends on who is calling: you read a handler's signature without knowing
// whether it is protected, and the protection vanishes the day someone moves
// the route out of the middleware group.
//
// Here, every protected operation starts with a line that NAMES its permission.
// Removing the protection requires deleting that line — visible on review.
type Guard struct {
	Module auth.Module
}

// Require demands a permission and returns the caller's identity.
//
// # The two calls are NOT merged
//
// `Verify` then `Authorize`: the token authenticates, it does not authorise. A
// single call doing both would bring permissions back within the token's scope
// through the back door, and the next natural optimisation would be to put them
// in there for good.
//
// # The order of the refusals matters
//
// An invalid token returns **401** before any question of permission: saying
// "permission denied" to someone who is not authenticated would teach them that
// the route exists and that a right guards it. A bearer who is authenticated
// but lacks the right gets **403** — and that distinction is useful, it spares
// them signing in over and over for a right they will not have any more than
// before.
func (g Guard) Require(
	ctx context.Context, header string, permission domain.Permission,
) (domain.Identity, error) {
	token, err := bearer(header)
	if err != nil {
		return domain.Identity{}, err
	}

	identity, err := g.Module.Verify(ctx, token)
	if err != nil {
		return domain.Identity{}, statusFor(err)
	}

	if err := g.Module.Authorize(ctx, identity.ID, permission); err != nil {
		return domain.Identity{}, statusFor(err)
	}
	return identity, nil
}

// mustPermission builds a permission at MOUNT time, not at call time.
//
// A malformed permission is a programming mistake, not a runtime condition: it
// depends on no input. Discovering it on the first protected call would return
// a 500 in production for a faulty string written months earlier — and the
// route would appear to work until then.
//
// The panic is therefore DELIBERATE and lives in an adapter, never in the core
// (rules/README.md: no `panic` in `domain/`, `ports/`, `application/`). It
// happens when the routes are mounted, that is at startup, that is before a
// single caller has been served.
func mustPermission(raw string) domain.Permission {
	permission, err := domain.NewPermission(raw)
	if err != nil {
		panic("malformed permission at mount time: " + raw + ": " + err.Error())
	}
	return permission
}
