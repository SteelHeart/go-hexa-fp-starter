// Package domain carries the business rules of registration, in the form of
// pure functions and immutable values.
//
// This package imports NEITHER transport, NOR persistence, NOR a logger. It does
// not read the clock and does not generate randomness: those effects are ports,
// injected. Enforced by arch-go.yml and depguard.
package domain

// ErrorCode enumerates the possible error outcomes of the feature.
//
// The set is CLOSED: every switch on ErrorCode is checked for exhaustiveness by
// the `exhaustive` linter. Adding a code therefore forces its translation to be
// handled in every surface — that is the intended effect.
type ErrorCode string

// The error codes of the feature.
const (
	CodeInvalidEmail       ErrorCode = "invalid_email"
	CodeWeakPassword       ErrorCode = "weak_password"
	CodeEmailAlreadyExists ErrorCode = "email_already_exists"
	CodeUnavailable        ErrorCode = "unavailable"
	CodeInternal           ErrorCode = "internal"
)

// Error is a business error. It is a VALUE, not an interface: the core depends
// on no open contract, and the set of errors stays enumerable.
type Error struct {
	// Code identifies the outcome. It is what surfaces translate.
	Code ErrorCode
	// Message is intended for the user: no technical detail, no sensitive
	// data.
	Message string
	// Field names the faulty field, for a validation error.
	Field string
	// cause carries the technical detail. Logged, NEVER returned to the
	// caller: an SQL error sent back to the client is a structure leak.
	cause error
}

// Ack reports a completed effect that has no value to return.
//
// It exists so that `ports/` contains NO structure declaration, not even the
// anonymous `struct{}` of `Result[struct{}, Error]` — the architecture rule
// refuses it, and it is right: a port must read like a signature, not like a
// type. The gain also shows at the call site, where `domain.Ack{}` reads better
// than `struct{}{}`.
type Ack struct{}

// NewError builds a business error.
func NewError(code ErrorCode, message string) Error {
	return Error{Code: code, Message: message}
}

// WithField states the faulty field and returns a new error.
func (e Error) WithField(field string) Error {
	e.Field = field
	return e
}

// WithCause attaches a technical detail and returns a new error.
func (e Error) WithCause(cause error) Error {
	e.cause = cause
	return e
}

// Cause exposes the technical detail, for logging only.
func (e Error) Cause() error { return e.cause }

// Error makes Error satisfy the error interface, which simplifies boundaries.
// The technical message does not appear in it.
func (e Error) Error() string {
	if e.Field != "" {
		return string(e.Code) + " (" + e.Field + "): " + e.Message
	}
	return string(e.Code) + ": " + e.Message
}
