// Package ports declares the contracts of authentication.
//
// This package contains ONLY type declarations: no struct, no function, no
// interface. A port is a function type — the smallest possible interface, so
// there is nothing to segregate (ADR 003).
//
// # Why `error` and not `Result[T, domain.Error]`
//
// `internal/core/**` uses `error`, `internal/modules/**` uses `Result`: the
// boundary is sharp and checkable. `auth` is the borderline case — it has a
// taxonomy that surfaces translate into 401, 403 and 422 — and that taxonomy
// goes through enumerated sentinels, recognisable by `errors.Is` (ADR 017).
//
// # The protocol, in three calls
//
//	token, err := authenticate(ctx, subject, secret)   // 401 if ErrInvalidCredentials
//	identity, err := verify(ctx, token)                // 401 if ErrTokenUnknown
//	err := authorize(ctx, identity, permission)        // 403 if ErrForbidden
//
// `verify` and `authorize` are TWO calls and not one, deliberately: the token
// authenticates, it does not authorise. Merging them would bring permissions
// back into the token through the back door.
package ports

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// ─── Primary ports: what the outside world may ask for ───────────────────────

// Authenticate exchanges a secret for a session.
//
// Errors: `domain.ErrIncomplete` if the request is malformed,
// `domain.ErrInvalidCredentials` otherwise — and NEVER a distinction between
// "unknown subject" and "wrong secret": the difference would tell an attacker
// which accounts exist.
type Authenticate = func(
	ctx context.Context,
	subject string,
	secret string,
) (domain.Session, error)

// Verify resolves a token into an identity.
//
// Errors: `domain.ErrTokenUnknown` for a token that is unknown, expired OR
// revoked — all three are conflated for the caller, and that is intended.
type Verify = func(ctx context.Context, token domain.Token) (domain.Identity, error)

// Authorize checks that an identity holds a permission.
//
// # The contract that carries the whole of ADR 017
//
// The implementation queries the PERSISTED state on every call. It does not
// read a claim carried by a token, and caches nothing unless that cache is an
// explicit and bounded decorator.
//
// The parameter is a `domain.Permission`, never a role: the compiler therefore
// forbids `authorize(ctx, id, "admin")`.
//
// Errors: `domain.ErrForbidden` if the permission is missing.
type Authorize = func(
	ctx context.Context,
	identity domain.IdentityID,
	permission domain.Permission,
) error

// Revoke invalidates a session immediately.
//
// Revoking an already unknown token is NOT an error: the operation is
// idempotent, because a client who signs out twice has done nothing wrong.
type Revoke = func(ctx context.Context, token domain.Token) error

// Register creates an identity and its secret.
//
// Errors: `domain.ErrSubjectTaken` if the subject already exists.
type Register = func(ctx context.Context, subject, secret string) (domain.Identity, error)

// DefineRole creates or replaces a role and its permissions.
//
// Without this port, no permission can ever be granted: `Authorize` would
// refuse everything, and the module would be inert while looking like it works.
//
// REPLACES rather than adds: removing a permission from a role must be as
// simple as adding one. An API that offered only addition would make removal be
// written by hand, hence badly.
type DefineRole = func(ctx context.Context, name string, permissions []string) error

// AssignRoles replaces an identity's roles.
//
// Errors: `domain.ErrInvalidCredentials` if the identity is unknown.
type AssignRoles = func(ctx context.Context, id domain.IdentityID, roles []string) error

// Deactivate closes an account, IMMEDIATELY.
//
// This is the gesture you make when you discover an account is compromised, and
// it is therefore the only moment where latency really matters. The identity
// stops authenticating, its already issued tokens stop being worth anything,
// and its permissions stop being granted — all three on the next call, without
// waiting for an expiry.
//
// Idempotent: deactivating an already closed account is not an error. Two
// administrators reacting to the same incident must not cancel each other out.
//
// Errors: `domain.ErrInvalidCredentials` if the identity is unknown.
type Deactivate = func(ctx context.Context, id domain.IdentityID) error

// Reactivate reopens a closed account.
//
// Exists because deactivation is sometimes a mistake, and a module that only
// knew how to close would have that repaired by hand in the store — hence
// badly, and without a trace.
//
// Errors: `domain.ErrInvalidCredentials` if the identity is unknown.
type Reactivate = func(ctx context.Context, id domain.IdentityID) error

// ─── Secondary ports: what the core needs from the world ─────────────────────

// SaveIdentity persists an identity and the digest of its secret.
//
// Error contract: `domain.ErrSubjectTaken` if the subject already exists. It is
// the implementation that decides, not the use case: between a check and a
// write there is a window that two simultaneous requests both cross.
type SaveIdentity = func(ctx context.Context, credential domain.Credential) error

// FindBySubject looks up an identity and its digest.
//
// Error contract: `domain.ErrInvalidCredentials` for an unknown subject — and
// not a "not found" error. The use case must be UNABLE to tell "this subject
// does not exist" from "the secret is wrong", otherwise it would end up saying
// so to the client.
type FindBySubject = func(ctx context.Context, subject domain.Subject) (domain.Credential, error)

// FindIdentity looks up an identity by its identifier.
type FindIdentity = func(ctx context.Context, id domain.IdentityID) (domain.Identity, error)

// SaveSession persists an issued session.
type SaveSession = func(ctx context.Context, session domain.Session) error

// FindSession looks up a session by its token.
//
// Error contract: `domain.ErrTokenUnknown`.
type FindSession = func(ctx context.Context, token domain.Token) (domain.Session, error)

// DeleteSession revokes a session. Idempotent.
type DeleteSession = func(ctx context.Context, token domain.Token) error

// SaveRole persists a role and its permissions.
type SaveRole = func(ctx context.Context, role domain.Role) error

// BindRoles replaces an identity's roles in the store.
type BindRoles = func(ctx context.Context, id domain.IdentityID, roles []string) error

// UpdateIdentity replaces the stored identity, without touching the secret's
// digest.
//
// The digest is NOT a parameter, deliberately: a signature that took it back
// would force every caller to carry it, hence to read it, hence to be able to
// log it. Closing an account has no reason to make a digest circulate.
//
// Error contract: `domain.ErrInvalidCredentials` if the identity is unknown.
type UpdateIdentity = func(ctx context.Context, identity domain.Identity) error

// Grants queries the PERSISTED state of an identity's permissions.
//
// Returns a boolean and not an error: "does not hold the permission" is not an
// outage, it is an answer. It is the use case that translates it into
// `domain.ErrForbidden`.
type Grants = func(ctx context.Context, id domain.IdentityID, permission domain.Permission) bool

// HashSecret produces the digest of a secret.
//
// It is a port PRECISELY because hashing is a costly and parameterised effect:
// the domain must neither choose it, nor tune it. The implementation comes from
// `internal/infrastructure/security` (Argon2id).
type HashSecret = func(plain string) (string, error)

// VerifySecret compares a secret against its digest.
//
// The implementation MUST compare in constant time. A comparison that stops at
// the first differing byte lets one measure how many characters are correct.
type VerifySecret = func(plain, encoded string) (bool, error)

// ─── Pure effect ports: the clock and the randomness ─────────────────────────

// Now returns the current instant.
//
// A port so that tests are deterministic: a session's expiry is checked by
// advancing a variable, not by waiting.
type Now = func() time.Time

// NewToken produces an opaque token.
//
// The implementation MUST draw from a cryptographically secure source. A
// predictable token is a bypassed authentication, and `math/rand` produces
// those.
type NewToken = func() (domain.Token, error)

// NewIdentityID produces an identity identifier.
type NewIdentityID = func() domain.IdentityID
