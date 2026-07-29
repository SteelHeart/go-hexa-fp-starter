// Package domain carries the vocabulary of authentication and authorisation,
// with no dependency.
//
// # Why this module exists
//
// A product evaluator asks two questions before any other: "how do I
// authenticate?" and "how do I isolate my customers?". As long as the first has
// no answer, no business module is deliverable — that is the P1 criterion in
// documentation/produit/personas.md, and it was worth "all of them" before this
// module.
//
// # This package is PURE
//
// No clock, no randomness, no I/O. The instant and the randomness enter through
// ports; passwords are never hashed here — hashing is a costly and
// parameterised effect, provided by internal/infrastructure/security.
package domain

import "errors"

// Sentinel errors of the module.
//
// # Why sentinels and not a Result[T, Error]
//
// `internal/core/**` returns `error`, `internal/modules/**` returns
// `Result[T, domain.Error]`: the boundary is sharp and checkable, and a core
// module is technical.
//
// `auth` is the borderline case of that invariant — it does have a taxonomy
// that surfaces must translate into 401, 403 and 422. That taxonomy therefore
// goes through ENUMERATED sentinels, recognisable by `errors.Is`. It is less
// expressive than a `Result`, and it is the accepted price of the core's
// homogeneity (ADR 017).
//
// # What each sentinel is worth to an HTTP surface
//
//	ErrInvalidCredentials  401 — and NEVER a distinction between "unknown
//	                             identifier" and "wrong password": the
//	                             difference tells an attacker which accounts
//	                             exist
//	ErrTokenUnknown        401 — token unknown, expired, or revoked. All three
//	                             are DELIBERATELY conflated on the client side
//	ErrForbidden           403 — authenticated, but the permission is missing
//	ErrIncomplete          422 — the request itself is malformed
var (
	// ErrInvalidCredentials refuses an authentication.
	//
	// A single message for "this subject does not exist" and "the secret is
	// wrong". Separating them would turn the sign-in form into an account
	// existence oracle — the most widespread mistake in this domain.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrTokenUnknown refuses a token.
	//
	// Unknown, expired or revoked: indistinguishable to the caller, and that is
	// intended. Saying "expired" rather than "unknown" confirms that a token
	// existed, hence that an account exists.
	ErrTokenUnknown = errors.New("unknown or expired token")

	// ErrForbidden refuses an action to an identity that is nonetheless
	// authenticated.
	ErrForbidden = errors.New("permission denied")

	// ErrIncomplete refuses a malformed request BEFORE any access to the store.
	//
	// The refusal happens upstream for two reasons: it costs no query, and it
	// does not let an empty string reach a driver, where it would become a
	// legitimate key.
	ErrIncomplete = errors.New("incomplete request")

	// ErrSubjectTaken refuses an already registered subject.
	ErrSubjectTaken = errors.New("this subject is already registered")
)
