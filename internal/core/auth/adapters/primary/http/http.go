// Package http exposes the authentication module on the HTTP surface.
//
// # A surface is a TRANSLATOR, never a place for business logic
//
// This package translates a request into a use case call, then a return value
// into a response. It validates nothing itself: the domain already does, and
// duplicating the validation guarantees that one day the two will diverge — to
// the detriment of the one that talks to the user.
//
// # File map
//
//	http.go             Mount, and the route map
//	open_session.go     POST   /v1/auth/sessions          — exchange a secret
//	close_session.go    DELETE /v1/auth/sessions/current  — revoke
//	identity.go         GET    /v1/auth/identity          — resolve the token
//	guard.go            require a permission — the only caller of Authorize
//	administration.go   the four PROTECTED routes
//	status.go           the translation of refusals into statuses, and the bearer
//
// # Two families of routes, and the boundary between them is a permission
//
// The three SESSION routes are public out of necessity: you cannot demand a
// token from someone who is coming to ask for one. The four ADMINISTRATION
// routes each require a distinct permission.
//
// ⚠️ This paragraph used to say "this surface does NOT expose administration",
// for want of a first administrator: publishing them without protecting them
// would have opened account creation to anyone. With bootstrapping decided
// (ADR 017 §6), they arrive — together with the guard that finally makes
// `Authorize` reachable.
package http

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// apiTag groups the module's operations in the served contract.
//
// A constant and not three literals: a grouping that diverges from one
// operation to the next shatters the section in the generated documentation,
// and nobody re-reads a contract to check a label.
const apiTag = "auth"

// Mount registers the module's operations on the API.
//
// Receives the Module, never a driver nor a store: a surface cannot bypass a
// use case, even by accident.
//
// The session operations are mounted together, deliberately. Mounting sign-in
// without revocation would ship a service you can get into without being able
// to get out — and revocation is the property ADR 017 buys.
//
// ⚠️ Said "the three operations" while four routes are mounted since the
// administration surface arrived. A count in a comment ages the day a route is
// added, and nothing says so.
func Mount(api huma.API, mod auth.Module) {
	mountOpenSession(api, mod)
	mountCloseSession(api, mod)
	mountIdentity(api, mod)
	mountAdministration(api, mod)
}
