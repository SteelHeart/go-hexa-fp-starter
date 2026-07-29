package domain

import (
	"fmt"
	"strings"
	"time"
)

// subjectMaxLen bounds the length of a subject.
//
// An explicit bound rather than none: without it, an input several megabytes
// long would cross the domain all the way to the store, where it would fail
// with a driver message instead of a business one.
const subjectMaxLen = 254

// IdentityID identifies an identity opaquely.
//
// A named type rather than a string: that is what prevents passing a subject
// where an identifier is expected, a confusion the compiler would never see
// between two `string`.
type IdentityID string

// Subject is what the user types to designate themselves — address, login,
// external identifier. Normalised and validated.
//
// The field is private: it is IMPOSSIBLE to build a non-normalised Subject
// outside this package. Without that, `Alice@X.COM` and `alice@x.com` would be
// TWO identities, and the second sign-in would fail without anyone
// understanding why.
type Subject struct{ value string }

// NewSubject normalises then validates a subject. The only construction path.
func NewSubject(raw string) (Subject, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))

	switch {
	case normalized == "":
		return Subject{}, fmt.Errorf("%w: the subject is mandatory", ErrIncomplete)
	case len(normalized) > subjectMaxLen:
		return Subject{}, fmt.Errorf("%w: the subject is too long", ErrIncomplete)
	case strings.ContainsAny(normalized, " \t\n"):
		return Subject{}, fmt.Errorf("%w: the subject cannot contain a space", ErrIncomplete)
	}
	return Subject{value: normalized}, nil
}

// String returns the normalised subject.
func (s Subject) String() string { return s.value }

// IsZero reports a subject that was never built.
func (s Subject) IsZero() bool { return s.value == "" }

// Masked returns a masked form, intended for logs.
//
// A subject is personal data: it is never logged in clear
// (rules/securite.md §5). And the authentication log is the one most readily
// exported to a third-party collector.
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

// Identity is an account known to the module.
//
// The secret's DIGEST does not appear in it, deliberately: it lives in the
// driver and never surfaces in a value that someone could log, serialise or
// return by mistake. This module is allowed to compare it, not to carry it
// around.
type Identity struct {
	ID        IdentityID
	Subject   Subject
	Roles     []string
	Active    bool
	CreatedAt time.Time
}

// NewIdentity builds a fresh identity, ACTIVE.
//
// The instant comes from the caller: the domain never reads the clock, and a
// test that read it would fail one day, for no reason.
//
// ⚠️ Unlike `user_registration` — whose account is born `pending` — an
// authentication identity is born active. The nuance is real: `auth` creates an
// identity ONLY on a request already authorised by its caller, whereas a public
// registration must be confirmed. Conflating the two would make either
// registration too permissive, or administration unworkable.
func NewIdentity(id IdentityID, subject Subject, roles []string, now time.Time) (Identity, error) {
	if id == "" {
		return Identity{}, fmt.Errorf("%w: the identifier is mandatory", ErrIncomplete)
	}
	if subject.IsZero() {
		return Identity{}, fmt.Errorf("%w: the subject is mandatory", ErrIncomplete)
	}

	return Identity{
		ID:        id,
		Subject:   subject,
		Roles:     normalizeRoles(roles),
		Active:    true,
		CreatedAt: now.UTC(),
	}, nil
}

// WithRoles returns a COPY carrying the given roles, NORMALISED.
//
// # The defect that normalising here fixes
//
// `NewRole` lowercases the role name, and `NewIdentity` did the same for the
// roles it received — but this method, taken by assignment, copied the names as
// they came. A role defined as `Comptable` was therefore kept under
// `comptable`, assigned under `Comptable`, and granted NOTHING.
//
// The fault produced no error at all: the administrator saw the role assigned,
// the person concerned got a 403, and nothing in the logs mentioned a case
// difference. Normalising in BOTH places is what makes the paths
// indistinguishable — leaving one out is enough to reopen the gap.
func (i Identity) WithRoles(roles []string) Identity {
	i.Roles = normalizeRoles(roles)
	return i
}

// normalizeRoles puts role names into their canonical form.
//
// The same one `NewRole` applies to the name it records: lowercase, without
// surrounding whitespace. Empty entries are dropped rather than refused — a
// blank line in an import grants nothing and does not deserve to fail the rest.
func normalizeRoles(roles []string) []string {
	kept := make([]string, 0, len(roles))
	for _, role := range roles {
		trimmed := strings.ToLower(strings.TrimSpace(role))
		if trimmed != "" {
			kept = append(kept, trimmed)
		}
	}
	return kept
}

// Deactivated returns a DEACTIVATED copy.
//
// A deactivated identity no longer authenticates and its tokens stop being
// worth anything — the driver is what applies that, but the value carries it.
func (i Identity) Deactivated() Identity {
	i.Active = false
	return i
}

// Reactivated returns a REACTIVATED copy.
//
// # Why two methods rather than one `SetActive(bool)`
//
// A boolean parameter cannot be read at the call site: `setActive(id, false)`
// forces you to look up the signature to know what happens, and
// `setActive(id, true)` slips through a review unnoticed. Two named methods
// make deactivation and its undoing visible where they are written — the same
// reason that split `SecurityHeaders(secure bool)` apart.
func (i Identity) Reactivated() Identity {
	i.Active = true
	return i
}
