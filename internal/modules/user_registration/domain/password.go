package domain

import (
	"strings"
	"unicode"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Length bounds of a password.
//
// The minimum is deliberately high and carries no composition requirement: a
// long passphrase resists better than a short "P@ssw0rd!", and composition rules
// push towards predictable passwords.
//
// The maximum protects the server: Argon2id on a 10 MB input is a denial of
// service handed over for free.
const (
	passwordMinLen = 12
	passwordMaxLen = 128
)

// RawPassword is a clear-text password, validated but not hashed.
//
// Its String() returns a marker: it becomes impossible to log it by accident,
// including through a %v on a struct that would contain it.
type RawPassword struct{ value string }

// NewRawPassword validates a clear-text password.
func NewRawPassword(raw string) result.Result[RawPassword, Error] {
	weak := func(reason string) result.Result[RawPassword, Error] {
		return result.Err[RawPassword, Error](
			NewError(CodeWeakPassword, reason).WithField("password"),
		)
	}

	switch {
	case len(raw) < passwordMinLen:
		return weak("le mot de passe doit faire au moins 12 caractères")
	case len(raw) > passwordMaxLen:
		return weak("le mot de passe ne peut pas dépasser 128 caractères")
	case strings.TrimSpace(raw) == "":
		return weak("le mot de passe ne peut pas être composé d'espaces")
	case distinctRunes(raw) < 5:
		return weak("le mot de passe est trop répétitif")
	}
	return result.Ok[RawPassword, Error](RawPassword{value: raw})
}

// Expose returns the clear-text password.
//
// Named this way deliberately: every call is visible in review, and there must
// be only one — the one made by the hashing port.
func (p RawPassword) Expose() string { return p.value }

// String masks the value. Prevents any accidental logging.
func (p RawPassword) String() string { return "[redacted]" }

// distinctRunes counts the distinct characters, ignoring case.
func distinctRunes(s string) int {
	seen := make(map[rune]struct{}, len(s))
	for _, r := range s {
		seen[unicode.ToLower(r)] = struct{}{}
	}
	return len(seen)
}

// PasswordHash is a password digest.
//
// It is not built by the domain: hashing is an effect, therefore a port. The
// domain merely carries the result around.
type PasswordHash struct{ value string }

// NewPasswordHash wraps a digest produced by an adapter.
func NewPasswordHash(encoded string) PasswordHash { return PasswordHash{value: encoded} }

// String returns the encoded digest.
func (h PasswordHash) String() string { return h.value }

// IsZero reports a digest that was never constructed.
func (h PasswordHash) IsZero() bool { return h.value == "" }
