package domain

import (
	"crypto/subtle"
	"fmt"
	"time"
)

// tokenMinLen bounds the length of an opaque token.
//
// 43 characters, that is 32 bytes in base64 without padding. Below that, the
// entropy becomes guessable; the bound is here so that the domain REFUSES a
// token built short, whatever port produced it.
const tokenMinLen = 43

// Token is an OPAQUE token — a random string, not a JWT.
//
// # Why opaque, and not signed (ADR 017 § 2 bis)
//
// The only real advantage of a signed token is validating WITHOUT touching the
// store. But `Authorize` goes there anyway, on every call: that is decision 1,
// and it is what makes revocation immediate. We would therefore pay for key
// management, key rotation, and the whole family of flaws specific to signed
// tokens — `none` algorithm accepted, HMAC/RSA confusion, expiry not checked —
// for a gain already given up.
//
// A random string compared against a record has none of those flaws.
type Token struct{ value string }

// NewToken validates a token produced by a randomness port.
func NewToken(raw string) (Token, error) {
	if len(raw) < tokenMinLen {
		return Token{}, fmt.Errorf(
			"%w: a token must be at least %d characters long", ErrIncomplete, tokenMinLen)
	}
	return Token{value: raw}, nil
}

// String returns the raw token. To be called only to hand it to the client or
// to the driver — never to log it.
func (t Token) String() string { return t.value }

// IsZero reports a token that was never built.
func (t Token) IsZero() bool { return t.value == "" }

// Equals compares two tokens in CONSTANT TIME.
//
// `==` stops at the first differing byte, so the duration of the comparison
// reveals how many leading characters are correct. That is the timing attack,
// and it is practicable on a local network. `subtle.ConstantTimeCompare` costs
// the same and removes the channel.
func (t Token) Equals(other Token) bool {
	return subtle.ConstantTimeCompare([]byte(t.value), []byte(other.value)) == 1
}

// Session is an issued token, attached to an identity and dated.
//
// Permissions do NOT appear in it, and that is the whole point of ADR 017: the
// token authenticates, it does not authorise. Putting them in would create a
// window during which a revoked access still works — and the day you revoke is
// the day you are in a hurry.
type Session struct {
	Token     Token
	Identity  IdentityID
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// NewSession builds a session bounded in time.
//
// A zero or negative duration is REFUSED: a session without expiry is an
// eternal session, and nobody decides that by accident.
func NewSession(token Token, identity IdentityID, now time.Time, ttl time.Duration) (Session, error) {
	switch {
	case token.IsZero():
		return Session{}, fmt.Errorf("%w: the token is mandatory", ErrIncomplete)
	case identity == "":
		return Session{}, fmt.Errorf("%w: the identity is mandatory", ErrIncomplete)
	case ttl <= 0:
		return Session{}, fmt.Errorf("%w: the session lifetime must be strictly positive", ErrIncomplete)
	}

	issued := now.UTC()
	return Session{
		Token:     token,
		Identity:  identity,
		IssuedAt:  issued,
		ExpiresAt: issued.Add(ttl),
	}, nil
}

// Expired reports whether the session has expired at the given instant.
//
// The instant is a PARAMETER: the domain does not read the clock. That is what
// makes it possible to test expiry without waiting.
//
// The bound is strict — a session expires AT its date, not after. A `>` would
// let the last millisecond through, which has no practical importance and makes
// the test depend on a comparison you have to read twice.
func (s Session) Expired(now time.Time) bool {
	return !now.UTC().Before(s.ExpiresAt)
}
