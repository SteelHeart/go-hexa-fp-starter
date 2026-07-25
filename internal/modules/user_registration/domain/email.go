package domain

import (
	"net/mail"
	"strings"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/result"
)

// emailMaxLen borne la longueur d'une adresse. La limite du protocole est 254
// octets ; au-delà, aucun serveur ne l'accepterait.
const emailMaxLen = 254

// Email est une adresse de courriel valide et normalisée.
//
// Le champ est non exporté : il est IMPOSSIBLE de fabriquer un Email invalide
// hors de ce paquet. Conséquence directe : une fonction qui reçoit un Email n'a
// plus à le valider — c'est le type qui le garantit, pas une convention.
type Email struct{ value string }

// NewEmail normalise puis valide une adresse. C'est le seul chemin de
// construction.
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

	// mail.ParseAddress accepte les formes « Nom <a@b.c> » : on refuse tout ce
	// qui ne serait pas exactement l'adresse, sinon deux entrées différentes
	// donneraient le même utilisateur.
	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Address != normalized || parsed.Name != "" {
		return invalid("l'adresse de courriel n'est pas valide")
	}
	if _, domain, found := strings.Cut(normalized, "@"); !found || !strings.Contains(domain, ".") {
		return invalid("le domaine de l'adresse est incomplet")
	}
	return result.Ok[Email, Error](Email{value: normalized})
}

// String retourne l'adresse normalisée.
func (e Email) String() string { return e.value }

// IsZero indique une adresse non construite.
func (e Email) IsZero() bool { return e.value == "" }

// Masked retourne une forme masquée, destinée aux journaux.
//
// Un courriel est une donnée personnelle : il ne se journalise jamais en clair
// (rules/securite.md §5).
func (e Email) Masked() string {
	local, domain, found := strings.Cut(e.value, "@")
	if !found || local == "" {
		return "***"
	}
	return local[:1] + "***@" + domain
}
