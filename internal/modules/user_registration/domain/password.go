package domain

import (
	"strings"
	"unicode"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// Bornes de longueur d'un mot de passe.
//
// Le minimum est volontairement haut et sans exigence de composition : une
// longue phrase de passe résiste mieux qu'un « P@ssw0rd! » court, et les règles
// de composition poussent à des mots de passe prévisibles.
//
// Le maximum protège le serveur : Argon2id sur une entrée de 10 Mo est un déni
// de service offert.
const (
	passwordMinLen = 12
	passwordMaxLen = 128
)

// RawPassword est un mot de passe en clair, validé mais non haché.
//
// Son String() retourne un marqueur : il devient impossible de le journaliser
// par accident, y compris via un %v sur une struct qui le contiendrait.
type RawPassword struct{ value string }

// NewRawPassword valide un mot de passe en clair.
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

// Expose retourne le mot de passe en clair.
//
// Nommée ainsi délibérément : tout appel est visible en revue, et il ne doit y
// en avoir qu'un — celui du port de hachage.
func (p RawPassword) Expose() string { return p.value }

// String masque la valeur. Empêche toute journalisation accidentelle.
func (p RawPassword) String() string { return "[redacted]" }

// distinctRunes compte les caractères distincts, en ignorant la casse.
func distinctRunes(s string) int {
	seen := make(map[rune]struct{}, len(s))
	for _, r := range s {
		seen[unicode.ToLower(r)] = struct{}{}
	}
	return len(seen)
}

// PasswordHash est un condensé de mot de passe.
//
// Il n'est pas construit par le domaine : le hachage est un effet, donc un port.
// Le domaine ne fait que transporter le résultat.
type PasswordHash struct{ value string }

// NewPasswordHash enveloppe un condensé produit par un adaptateur.
func NewPasswordHash(encoded string) PasswordHash { return PasswordHash{value: encoded} }

// String retourne le condensé encodé.
func (h PasswordHash) String() string { return h.value }

// IsZero indique un condensé non construit.
func (h PasswordHash) IsZero() bool { return h.value == "" }
