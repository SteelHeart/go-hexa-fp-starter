// Package auth is the PUBLISHED LANGUAGE of the authentication module.
//
// # What this package holds, and what it will never hold
//
// Serialisable primitive types and routes. Never a domain type, never a rule,
// never a data access. `domain.Token`, `domain.Subject` and `domain.Permission`
// all have a PRIVATE field: publishing them here would make them forgeable from
// the outside, and the normalisation they guarantee would stop being a
// guarantee.
//
// # What is deliberately absent
//
// No shape carries a PERMISSION. That is decision 1 of ADR 017 seen from the
// contract: the token authenticates, it does not authorise. Publishing a session
// response listing the rights would invite every consumer to cache them, and
// revocation would stop being immediate without a single line of this repository
// changing.
//
// # Versioning
//
// A contract is immutable. A breaking change creates a `V2` beside the `V1`.
package auth

import "time"

// ModuleName identifies the owning module.
const ModuleName = "auth"

// SchemaName is the module's Postgres schema.
//
// `platform` rather than a dedicated schema: `auth` is a CORE module, and core
// modules share the platform schema (ADR 011). One schema per core module would
// multiply roles without isolating anything more — they all belong to the
// starter.
const SchemaName = "platform"

// SessionRequest is the published shape of a session request.
//
// `subject` rather than `email`: the module makes no assumption about what
// designates an account. An address today, an external identifier tomorrow,
// without changing the contract.
type SessionRequest struct {
	Subject string `json:"subject"`
	Secret  string `json:"secret"`
}

// SessionResponse is the published shape of an open session.
//
// # Three fields, and not one more
//
// No roles, no permissions, no digest. A client gets what it needs to
// authenticate and what it needs to know when to start over — nothing inviting
// it to decide for itself what it is allowed to do.
//
// `expires_at` is INFORMATION, not a guarantee: the session may stop being valid
// earlier, through revocation or account closure. A client relying on it to
// avoid handling a 401 would be wrong exactly when it matters.
type SessionResponse struct {
	Token      string    `json:"token"`
	IdentityID string    `json:"identity_id"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// IdentityResponse is the published shape of a resolved identity.
//
// ROLES appear, permissions do not. A role is an administrative label — useful
// to display "accountant" in a user interface — whereas a permission is a
// decision, and a decision is asked for, not read from a cached response.
type IdentityResponse struct {
	IdentityID string    `json:"identity_id"`
	Subject    string    `json:"subject"`
	Roles      []string  `json:"roles"`
	CreatedAt  time.Time `json:"created_at"`
}

// Permissions the administration surface requires.
//
// # Why they are PUBLISHED rather than private to the module
//
// A permission is DATA: it is granted inside a role, in the database, without a
// deployment (ADR 017 §4). An application mounting this starter must therefore
// be able to name them to compose its own roles — otherwise it would have to
// copy string literals, and a typo would silently grant nothing.
//
// The shape is the one the domain imposes: `domain.resource.action`, lowercase,
// exactly three segments.
const (
	// PermissionIdentityCreate allows creating an identity.
	PermissionIdentityCreate = "auth.identity.create"

	// PermissionIdentityRoles allows assigning roles.
	PermissionIdentityRoles = "auth.identity.roles"

	// PermissionIdentityClose allows closing and reopening an account. ONE
	// permission for both directions: whoever can close can reopen, and
	// separating them would produce a state where one closes without being able
	// to undo.
	PermissionIdentityClose = "auth.identity.close"

	// PermissionRoleWrite allows defining a role and its permissions.
	//
	// ⚠️ This is the most powerful permission of the module: whoever holds it can
	// grant themselves all the others. It is named apart so that this fact is
	// visible when composing a role, rather than discovered during an audit.
	PermissionRoleWrite = "auth.role.write"
)

// Published administration shapes.
type (
	// CreateIdentityRequest creates an account.
	CreateIdentityRequest struct {
		Subject string `json:"subject"`
		Secret  string `json:"secret"`
	}

	// DefineRoleRequest replaces a role and its permissions.
	//
	// REPLACES rather than adds: removing a permission must be as simple as
	// adding one. An API offering only addition would have removal written by
	// hand, hence badly.
	DefineRoleRequest struct {
		Permissions []string `json:"permissions"`
	}

	// AssignRolesRequest replaces an identity's roles.
	AssignRolesRequest struct {
		Roles []string `json:"roles"`
	}
)

// Routes exposed by the module's HTTP surface.
//
// Assumed globals: these are constants of the published language, and Go has no
// structured constant. Turning them into functions would disguise data as
// computation without protecting anything, since the returned value would be
// copyable anyway.
//
//nolint:gochecknoglobals // published-language constants: Go has no structured constant
var (
	// OpenSessionRoute exchanges a secret for a token.
	//
	// `POST /v1/auth/sessions` rather than `/login`: the resource is the SESSION,
	// and creating it is a POST. That is what makes closing natural — a DELETE on
	// the same resource — instead of a second invented verb.
	OpenSessionRoute = struct {
		Method string
		Path   string
	}{Method: "POST", Path: "/v1/auth/sessions"}

	// CloseSessionRoute revokes the presented token.
	CloseSessionRoute = struct {
		Method string
		Path   string
	}{Method: "DELETE", Path: "/v1/auth/sessions/current"}

	// IdentityRoute resolves the presented token into an identity.
	IdentityRoute = struct {
		Method string
		Path   string
	}{Method: "GET", Path: "/v1/auth/identity"}

	// CreateIdentityRoute creates an account. PROTECTED.
	CreateIdentityRoute = struct {
		Method string
		Path   string
	}{Method: "POST", Path: "/v1/auth/identities"}

	// DefineRoleRoute defines a role. PROTECTED.
	DefineRoleRoute = struct {
		Method string
		Path   string
	}{Method: "PUT", Path: "/v1/auth/roles/{name}"}

	// AssignRolesRoute assigns roles to an identity. PROTECTED.
	AssignRolesRoute = struct {
		Method string
		Path   string
	}{Method: "PUT", Path: "/v1/auth/identities/{id}/roles"}

	// CloseIdentityRoute closes an account, IMMEDIATELY. PROTECTED.
	CloseIdentityRoute = struct {
		Method string
		Path   string
	}{Method: "DELETE", Path: "/v1/auth/identities/{id}"}
)
