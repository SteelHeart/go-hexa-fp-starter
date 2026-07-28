package domain

import (
	"fmt"
	"strings"
)

// recipientMaxLen borne la longueur d'une adresse.
//
// 254 octets, la limite d'une adresse de courriel selon la RFC 5321.
const recipientMaxLen = 254

// Recipient est l'adresse d'un destinataire, normalisée et validée.
//
// # Le champ est privé, et c'est la garantie
//
// Il est IMPOSSIBLE de fabriquer un destinataire non normalisé hors de ce
// paquet. Sans cela, `Alice@X.COM` et `alice@x.com` seraient deux destinataires,
// et une déduplication d'envois laisserait passer le doublon.
//
// # Pourquoi ce type existe alors que `auth` et `user_registration` en ont un
//
// Parce qu'un module noyau ne peut importer ni l'un ni l'autre : `internal/core`
// ne connaît aucun module métier (arch-go), et deux modules noyau ne se partagent
// pas un domaine. La répétition est le prix de l'indépendance — c'est elle qui
// permettra d'extraire `internal/core` en module Go séparé (ADR 012).
type Recipient struct{ value string }

// NewRecipient normalise puis valide une adresse.
//
// # La validation est délibérément MINIMALE
//
// Non vide, une seule arobase, du texte de part et d'autre, pas d'espace. Aucune
// expression rationnelle « conforme RFC 5322 » : elles rejettent des adresses
// valides — les apostrophes, les caractères accentués, les sous-adresses `+` —
// et un destinataire refusé à tort ne reçoit jamais rien, sans que personne s'en
// aperçoive. Le seul verdict fiable sur une adresse est un envoi réussi.
func NewRecipient(raw string) (Recipient, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))

	local, domain, found := strings.Cut(normalized, "@")
	switch {
	case normalized == "":
		return Recipient{}, fmt.Errorf("%w: le destinataire est obligatoire", ErrIncomplete)
	case len(normalized) > recipientMaxLen:
		return Recipient{}, fmt.Errorf("%w: l'adresse dépasse %d octets", ErrIncomplete, recipientMaxLen)
	case strings.ContainsAny(normalized, " \t\n"):
		return Recipient{}, fmt.Errorf("%w: l'adresse ne peut pas contenir d'espace", ErrIncomplete)
	case !found || local == "" || domain == "":
		return Recipient{}, fmt.Errorf("%w: adresse %q — attendu `local@domaine`", ErrIncomplete, raw)
	case strings.Contains(domain, "@"):
		return Recipient{}, fmt.Errorf("%w: adresse %q — une seule arobase", ErrIncomplete, raw)
	}

	return Recipient{value: normalized}, nil
}

// String rend l'adresse normalisée. À n'appeler que pour la remettre au
// fournisseur — jamais pour la journaliser.
func (r Recipient) String() string { return r.value }

// IsZero indique un destinataire non construit.
func (r Recipient) IsZero() bool { return r.value == "" }

// Masked rend une forme masquée, destinée aux journaux.
//
// Une adresse est une donnée personnelle : elle ne se journalise jamais en clair
// (rules/securite.md §5). Le domaine reste lisible parce qu'il sert au
// diagnostic — « tous les échecs vont vers le même domaine » est l'information
// qu'on cherche pendant un incident — et parce qu'il n'identifie personne.
func (r Recipient) Masked() string {
	local, domain, found := strings.Cut(r.value, "@")
	if !found || local == "" {
		return "***"
	}
	return local[:1] + "***@" + domain
}
