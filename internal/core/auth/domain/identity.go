package domain

import (
	"fmt"
	"strings"
	"time"
)

// subjectMaxLen borne la longueur d'un sujet.
//
// Une borne explicite plutôt qu'aucune : sans elle, une entrée de plusieurs
// mégaoctets traverserait le domaine jusqu'au magasin, où elle échouerait avec un
// message de pilote au lieu d'un message métier.
const subjectMaxLen = 254

// IdentityID identifie une identité de façon opaque.
//
// Un type nommé plutôt qu'une chaîne : c'est ce qui empêche de passer un sujet là
// où un identifiant est attendu, confusion que le compilateur ne verrait jamais
// entre deux `string`.
type IdentityID string

// Subject est ce que l'utilisateur saisit pour se désigner — adresse, login,
// identifiant externe. Normalisé et validé.
//
// Le champ est privé : il est IMPOSSIBLE de fabriquer un Subject non normalisé
// hors de ce paquet. Sans cela, `Alice@X.COM` et `alice@x.com` seraient DEUX
// identités, et la seconde connexion échouerait sans que personne comprenne.
type Subject struct{ value string }

// NewSubject normalise puis valide un sujet. Seul chemin de construction.
func NewSubject(raw string) (Subject, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))

	switch {
	case normalized == "":
		return Subject{}, fmt.Errorf("%w: le sujet est obligatoire", ErrIncomplete)
	case len(normalized) > subjectMaxLen:
		return Subject{}, fmt.Errorf("%w: le sujet est trop long", ErrIncomplete)
	case strings.ContainsAny(normalized, " \t\n"):
		return Subject{}, fmt.Errorf("%w: le sujet ne peut pas contenir d'espace", ErrIncomplete)
	}
	return Subject{value: normalized}, nil
}

// String rend le sujet normalisé.
func (s Subject) String() string { return s.value }

// IsZero indique un sujet non construit.
func (s Subject) IsZero() bool { return s.value == "" }

// Masked rend une forme masquée, destinée aux journaux.
//
// Un sujet est une donnée personnelle : il ne se journalise jamais en clair
// (rules/securite.md §5). Et c'est le journal d'authentification qu'on exporte
// le plus volontiers vers un collecteur tiers.
func (s Subject) Masked() string {
	if len(s.value) <= 2 {
		return "***"
	}
	local, domain, found := strings.Cut(s.value, "@")
	if found && local != "" {
		return local[:1] + "***@" + domain
	}
	return s.value[:1] + "***"
}

// Identity est un compte connu du module.
//
// Le CONDENSÉ du secret n'y figure pas, délibérément : il vit dans le pilote et
// ne remonte jamais dans une valeur que quelqu'un pourrait journaliser, sérialiser
// ou renvoyer par erreur. Ce module a le droit de le comparer, pas de le promener.
type Identity struct {
	ID        IdentityID
	Subject   Subject
	Roles     []string
	Active    bool
	CreatedAt time.Time
}

// NewIdentity construit une identité neuve, ACTIVE.
//
// L'instant vient de l'appelant : le domaine ne lit jamais l'horloge, et un test
// qui la lirait échouerait un jour, sans raison.
//
// ⚠️ Contrairement à `user_registration` — dont le compte naît `pending` — une
// identité d'authentification naît active. La nuance est réelle : `auth` ne crée
// une identité QUE sur une demande déjà autorisée par son appelant, alors qu'une
// inscription publique doit être confirmée. Confondre les deux rendrait soit
// l'inscription trop permissive, soit l'administration impraticable.
func NewIdentity(id IdentityID, subject Subject, roles []string, now time.Time) (Identity, error) {
	if id == "" {
		return Identity{}, fmt.Errorf("%w: l'identifiant est obligatoire", ErrIncomplete)
	}
	if subject.IsZero() {
		return Identity{}, fmt.Errorf("%w: le sujet est obligatoire", ErrIncomplete)
	}

	kept := make([]string, 0, len(roles))
	for _, role := range roles {
		trimmed := strings.ToLower(strings.TrimSpace(role))
		if trimmed != "" {
			kept = append(kept, trimmed)
		}
	}

	return Identity{
		ID:        id,
		Subject:   subject,
		Roles:     kept,
		Active:    true,
		CreatedAt: now.UTC(),
	}, nil
}

// WithRoles retourne une COPIE portant les rôles donnés.
func (i Identity) WithRoles(roles []string) Identity {
	i.Roles = append([]string(nil), roles...)
	return i
}

// Deactivated retourne une COPIE désactivée.
//
// Une identité désactivée ne s'authentifie plus et ses jetons cessent de valoir —
// c'est le pilote qui l'applique, mais la valeur le porte.
func (i Identity) Deactivated() Identity {
	i.Active = false
	return i
}
