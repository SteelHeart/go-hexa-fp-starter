package domain

import (
	"net/mail"
	"strings"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// emailMaxLen bounds the length of an email address. The protocol limit is 254
// bytes; beyond that, no server would accept it.
const emailMaxLen = 254

// Email is a valid, normalised email address.
//
// The field is unexported: it is IMPOSSIBLE to build an invalid Email outside
// this package. Direct consequence: a function that receives an Email no longer
// has to validate it — the type guarantees it, not a convention.
type Email struct{ value string }

// NewEmail normalises then validates an email address. This is the only
// construction path.
func NewEmail(raw string) result.Result[Email, Error] {
	normalized := strings.ToLower(strings.TrimSpace(raw))

	invalid := func(reason string) result.Result[Email, Error] {
		return result.Err[Email, Error](
			NewError(CodeInvalidEmail, reason).WithField("email"),
		)
	}

	switch {
	case normalized == "":
		return invalid("l'adresse de courriel est obligatoire")
	case len(normalized) > emailMaxLen:
		return invalid("l'adresse de courriel est trop longue")
	}

	// mail.ParseAddress accepts the "Name <a@b.c>" forms: we refuse anything
	// that would not be exactly the address, otherwise two different inputs
	// would yield the same user.
	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Address != normalized || parsed.Name != "" {
		return invalid("l'adresse de courriel n'est pas valide")
	}
	if _, domain, found := strings.Cut(normalized, "@"); !found || !strings.Contains(domain, ".") {
		return invalid("le domaine de l'adresse est incomplet")
	}
	return result.Ok[Email, Error](Email{value: normalized})
}

// String returns the normalised address.
func (e Email) String() string { return e.value }

// IsZero reports an address that was never constructed.
func (e Email) IsZero() bool { return e.value == "" }

// Masked returns a masked form, intended for logs.
//
// An email address is personal data: it is never logged in clear
// (rules/securite.md §5).
func (e Email) Masked() string {
	local, domain, found := strings.Cut(e.value, "@")
	if !found || local == "" {
		return "***"
	}
	return local[:1] + "***@" + domain
}
