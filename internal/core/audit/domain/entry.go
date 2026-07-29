// Package domain carries the vocabulary of the audit log, with no dependency.
package domain

import (
	"errors"
	"time"
)

// ErrIncomplete refuses an entry without an actor, an action or an entity.
//
// Declared in the domain and not in a driver: every driver must refuse the
// SAME thing, and a caller must be able to recognise it with errors.Is
// whichever driver is plugged in. This is the operational form of
// substitutability (ADR 003).
var ErrIncomplete = errors.New("incomplete audit entry")

// Entry is an audit fact.
//
// Deliberately poor: an audit log answers "who did what, when, on what". It
// does not answer "why" — that is the job of the issue and of the PR.
type Entry struct {
	// Actor identifies the author. An identifier, never a name nor an address.
	Actor string
	// Action names the fact, in the past tense and in business form:
	// "user.registered", not "INSERT users".
	Action     string
	EntityType string
	EntityID   string
	// Metadata must contain NO personal data in the clear: the log is kept for
	// a long time and read by humans (rules/securite.md §5).
	Metadata map[string]any
	At       time.Time
}

// WithTime returns a timestamped copy. The instant comes from a port, never
// from time.Now(): an audit test must be deterministic.
func (e Entry) WithTime(at time.Time) Entry {
	e.At = at.UTC()
	return e
}

// IsComplete says whether the entry carries the usable minimum.
//
// An incomplete audit entry is worse than an absent one: it gives the illusion
// of traceability. The driver therefore refuses to write what is not complete.
func (e Entry) IsComplete() bool {
	return e.Actor != "" && e.Action != "" && e.EntityType != "" && e.EntityID != ""
}
