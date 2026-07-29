// Package userregistration is the PUBLISHED LANGUAGE of the registration module.
//
// # Why this package lives outside the business modules
//
// A business module NEVER imports another business module (ADR 001): that rule
// is absolute and enforced by arch-go. Yet a module sometimes needs a capability
// of another one. The published contract resolves the contradiction: it is
// physically separate from the modules, holds only primitive types, and exposes
// NOTHING of the internal domain.
//
// What this package may hold: event names, serialisable payloads, command and
// response shapes.
//
// What it will never hold: a domain type, a business rule, a data access. The
// producing module remains the sole owner of its tables.
//
// # Versioning
//
// A contract is immutable. A breaking change creates a `V2` beside the `V1`:
// consumers are deployed independently and still read the `V1`.
package userregistration

import "time"

// ModuleName identifies the owning module. Serves as the transport
// configuration key and as the Postgres schema name.
const ModuleName = "user_registration"

// SchemaName is the module's exclusive Postgres schema.
//
// No other module may access it, not even for reading. The CI `isolation` guard
// enforces it (ADR 011).
const SchemaName = "user_registration"

// EventUserRegisteredV1 names the published registration event.
const EventUserRegisteredV1 = "user.registered.v1"

// UserRegisteredV1 is the payload published when a user registers.
//
// Primitive types only: a consumer written in another language, or deployed
// separately, must be able to read it.
type UserRegisteredV1 struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	RegisteredAt time.Time `json:"registered_at"`
}

// RegisterRequest is the published shape of the registration command.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterResponse is the published shape of the registration result.
//
// No password digest, no internal field: what leaves the module is what another
// module is entitled to know, nothing more.
type RegisterResponse struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRoute is the HTTP route exposing the capability, used when the module
// is called remotely.
//
// Assumed global: it is a CONSTANT of the published language, just like the types
// above. Go has no structured constant; turning it into a function
// (`func RegisterRoute() ...`) would disguise data as computation without
// protecting anything, since the returned value would be copyable and mutable
// anyway.
//
//nolint:gochecknoglobals // published-language constant: Go has no structured constant
var RegisterRoute = struct {
	Method string
	Path   string
}{Method: "POST", Path: "/v1/users"}
