package domain

import "time"

// UserID identifies a user. A dedicated type, never a bare string in a signature
// (rules/ports-et-contrats.md §3).
type UserID string

// String returns the identifier.
func (id UserID) String() string { return string(id) }

// IsZero reports an identifier that was never set.
func (id UserID) IsZero() bool { return id == "" }

// Status enumerates the states of an account. Closed set, exhaustive switch.
type Status string

// The states of an account.
const (
	StatusPending Status = "pending"
	StatusActive  Status = "active"
	StatusBlocked Status = "blocked"
)

// User is a registered user. All its fields are domain types: it is structurally
// impossible to place an invalid value in it.
type User struct {
	ID           UserID
	Email        Email
	PasswordHash PasswordHash
	Status       Status
	CreatedAt    time.Time
}

// NewUser builds a newly registered user.
//
// The account is born PENDING: a confirmation email will have to activate it.
// Being born active would be a "fail-open" — the address is not proven yet.
func NewUser(id UserID, email Email, hash PasswordHash, at time.Time) User {
	return User{
		ID:           id,
		Email:        email,
		PasswordHash: hash,
		Status:       StatusPending,
		CreatedAt:    at.UTC(),
	}
}

// WithStatus returns a copy carrying the new state. The original is never
// modified: the receiver is a value, and `revive` would refuse a pointer.
func (u User) WithStatus(status Status) User {
	u.Status = status
	return u
}

// CanAuthenticate reports whether the account may open a session.
//
// Deny by default: every unknown state refuses. Adding a Status without
// completing this switch fails the CI (`exhaustive`).
func (u User) CanAuthenticate() bool {
	switch u.Status {
	case StatusActive:
		return true
	case StatusPending, StatusBlocked:
		return false
	default:
		return false
	}
}
