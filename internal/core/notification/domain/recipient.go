package domain

import (
	"fmt"
	"strings"
)

// recipientMaxLen bounds the length of an address.
//
// 254 bytes, the limit of an email address according to RFC 5321.
const recipientMaxLen = 254

// Recipient is a recipient's address, normalised and validated.
//
// # The field is private, and that is the guarantee
//
// It is IMPOSSIBLE to build a non-normalised recipient outside this package.
// Without that, `Alice@X.COM` and `alice@x.com` would be two recipients, and a
// send deduplication would let the duplicate through.
//
// # Why this type exists although `auth` and `user_registration` have one
//
// Because a core module can import neither: `internal/core` knows no business
// module (arch-go), and two core modules do not share a domain. The repetition
// is the price of independence — it is what will allow extracting
// `internal/core` into a separate Go module (ADR 012).
type Recipient struct{ value string }

// NewRecipient normalises then validates an address.
//
// # The validation is deliberately MINIMAL
//
// Non-empty, a single at-sign, text on both sides, no whitespace. No
// "RFC 5322 compliant" regular expression: they reject valid addresses —
// apostrophes, accented characters, `+` sub-addresses — and a recipient wrongly
// refused never receives anything, without anyone noticing. The only reliable
// verdict on an address is a successful send.
func NewRecipient(raw string) (Recipient, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))

	local, domain, found := strings.Cut(normalized, "@")
	switch {
	case normalized == "":
		return Recipient{}, fmt.Errorf("%w: the recipient is required", ErrIncomplete)
	case len(normalized) > recipientMaxLen:
		return Recipient{}, fmt.Errorf("%w: the address exceeds %d bytes", ErrIncomplete, recipientMaxLen)
	case strings.ContainsAny(normalized, " \t\n"):
		return Recipient{}, fmt.Errorf("%w: the address cannot contain whitespace", ErrIncomplete)
	case !found || local == "" || domain == "":
		return Recipient{}, fmt.Errorf("%w: address %q — expected `local@domain`", ErrIncomplete, raw)
	case strings.Contains(domain, "@"):
		return Recipient{}, fmt.Errorf("%w: address %q — a single at-sign", ErrIncomplete, raw)
	}

	return Recipient{value: normalized}, nil
}

// String returns the normalised address. To be called only to hand it over to
// the provider — never to log it.
func (r Recipient) String() string { return r.value }

// IsZero reports a recipient that was never built.
func (r Recipient) IsZero() bool { return r.value == "" }

// Masked returns a masked form, intended for logs.
//
// An address is personal data: it is never logged in clear
// (rules/securite.md §5). The domain stays readable because it serves diagnosis
// — "every failure goes towards the same domain" is the information one looks
// for during an incident — and because it identifies nobody.
func (r Recipient) Masked() string {
	local, domain, found := strings.Cut(r.value, "@")
	if !found || local == "" {
		return "***"
	}
	return local[:1] + "***@" + domain
}
