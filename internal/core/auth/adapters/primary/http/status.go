package http

import (
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/domain"
)

// bearerPrefix is the expected authentication scheme.
const bearerPrefix = "Bearer "

// bearer extracts the token from an `Authorization` header.
//
// # Every failure returns 401, never 422
//
// Missing header, unknown scheme, token too short: a single status.
// Distinguishing "malformed token" from "unknown token" would tell an attacker
// that their string has the right SHAPE — hence that they are getting close —
// and that is exactly the hint one does not want to give about a credential.
//
// The prefix is compared case-insensitively: RFC 7235 declares the scheme
// case-insensitive, and refusing `bearer ` would fail perfectly correct clients
// with a message that would blame the token.
func bearer(header string) (domain.Token, error) {
	raw := strings.TrimSpace(header)
	if len(raw) <= len(bearerPrefix) || !strings.EqualFold(raw[:len(bearerPrefix)], bearerPrefix) {
		return domain.Token{}, errBadToken
	}

	token, err := domain.NewToken(strings.TrimSpace(raw[len(bearerPrefix):]))
	if err != nil {
		return domain.Token{}, errBadToken
	}
	return token, nil
}

// The two 401 refusals, separated by ROUTE and not by cause.
//
// # The defect this separation fixes
//
// The first version returned "authentication required" for a malformed token
// and "invalid credentials" for a well-formed but unknown token — both on the
// SAME route. The gap told an attacker that their string had the right shape,
// hence that they were getting close: precisely the hint a single 401 sets out
// to remove. Found by the test, not by review.
//
// The split holds because the two sentinels never cross: `Verify` only returns
// `ErrTokenUnknown`, `Authenticate` only `ErrInvalidCredentials`. Each route
// therefore has ONE single message, and each reads naturally where it appears.
var (
	// errBadToken covers everything to do with the TOKEN — missing, malformed,
	// unknown, expired, revoked.
	errBadToken = huma.Error401Unauthorized("authentication required")

	// errBadCredentials covers everything to do with the SECRET — unknown
	// subject, malformed subject, wrong secret, closed account.
	errBadCredentials = huma.Error401Unauthorized("invalid credentials")
)

// statusFor translates a module refusal into an HTTP response.
//
// # Why `errors.Is` and not an exhaustive `switch`
//
// `internal/core/**` returns an `error`, not a `Result[T, domain.Error]`: there
// is therefore no enumerable code for `exhaustive` to watch over, as it does
// for `user_registration`. The price of the core's homogeneity is written down
// here rather than kept quiet: adding a sentinel to the domain without adding
// it to this list would silently make it fall through to a 500.
//
// The fallback IS a 500 — never a 200, never a "default" 403. The unknown is
// treated as an outage, not as an acceptable request.
func statusFor(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		// A single message for "this account does not exist", "the secret is
		// wrong" and "the account is closed". The distinction is precisely what
		// an attacker is looking for.
		return errBadCredentials

	case errors.Is(err, domain.ErrTokenUnknown):
		// The SAME message `bearer` returns on a malformed token: on a protected
		// route, everything to do with the token looks alike.
		return errBadToken

	case errors.Is(err, domain.ErrForbidden):
		// 403 and not 401: the caller is authenticated. Returning 401 would push
		// them to sign in over and over for a right they will not have any more
		// than before.
		return huma.Error403Forbidden("permission denied")

	case errors.Is(err, domain.ErrSubjectTaken):
		// 409 and not 422: the request is well formed, it is the server's STATE
		// that stands in its way.
		return huma.Error409Conflict("this subject is already registered")

	case errors.Is(err, domain.ErrIncomplete):
		return huma.Error422UnprocessableEntity(err.Error())

	case errors.Is(err, auth.ErrDisabled):
		// 503 and not 501: the capability EXISTS, it is not enabled on this
		// deployment. That is an operational decision, not a missing feature,
		// and 503 is what allows a client to retry later.
		return huma.Error503ServiceUnavailable("authentication unavailable")

	default:
		// The technical cause is NEVER returned to the caller: it is logged by
		// the middleware, tied to the correlation identifier.
		return huma.Error500InternalServerError("an internal error occurred")
	}
}
