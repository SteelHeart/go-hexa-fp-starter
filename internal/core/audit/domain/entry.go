// Package domain porte le vocabulaire du journal d'audit, sans dépendance.
package domain

import (
	"errors"
	"time"
)

// ErrIncomplete refuse une entrée sans acteur, action ou entité.
//
// Déclarée dans le domaine et non dans un pilote : tous les pilotes doivent
// refuser la MÊME chose, et un appelant doit pouvoir la reconnaître avec
// errors.Is quel que soit le pilote branché. C'est la forme opérationnelle de
// la substituabilité (ADR 003).
var ErrIncomplete = errors.New("entrée d'audit incomplète")

// Entry est un fait d'audit.
//
// Volontairement pauvre : un journal d'audit répond à « qui a fait quoi, quand,
// sur quoi ». Il ne répond pas à « pourquoi » — ça, c'est le rôle de l'issue et
// de la PR.
type Entry struct {
	// Actor identifie l'auteur. Un identifiant, jamais un nom ni une adresse.
	Actor string
	// Action nomme le fait, au passé et à la forme métier : "user.registered",
	// pas "INSERT users".
	Action     string
	EntityType string
	EntityID   string
	// Metadata ne doit contenir AUCUNE donnée personnelle en clair : le journal
	// est conservé longtemps et lu par des humains (rules/securite.md §5).
	Metadata map[string]any
	At       time.Time
}

// WithTime retourne une copie horodatée. L'instant vient d'un port, jamais de
// time.Now() : un test d'audit doit être déterministe.
func (e Entry) WithTime(at time.Time) Entry {
	e.At = at.UTC()
	return e
}

// IsComplete indique si l'entrée porte le minimum exploitable.
//
// Une entrée d'audit incomplète est pire qu'absente : elle donne l'illusion
// d'une traçabilité. Le pilote refuse donc d'écrire ce qui n'est pas complet.
func (e Entry) IsComplete() bool {
	return e.Actor != "" && e.Action != "" && e.EntityType != "" && e.EntityID != ""
}
