// Package domain carries the vocabulary of the outbox, without any dependency.
//
// It imports neither driver, nor database driver, nor identifier generator:
// the identifier is supplied by the driver, which makes this package testable
// with nothing and verifiable by arch-go.yml.
package domain

import "time"

// MessageID identifies an outbox message.
type MessageID string

// String returns the identifier.
func (id MessageID) String() string { return string(id) }

// Status enumerates the states of a message. A closed set: every switch is
// checked for exhaustiveness by the linter.
type Status string

// The states of a message.
const (
	// StatusPending is awaiting its publication, or its next attempt.
	StatusPending Status = "pending"
	// StatusDone has been published successfully.
	StatusDone Status = "done"
	// StatusFailed has exhausted its attempts. NEVER deleted: it is the only
	// trace of what has not been published.
	StatusFailed Status = "failed"
)

// NewMessage is an intent to publish, before persistence.
//
// The identifier and the timestamp do not appear in it: those are effects,
// produced by the driver. That is what makes a test deterministic.
type NewMessage struct {
	Type        string
	AggregateID string
	// Payload is opaque JSON. The domain does not serialise it itself: a
	// consumer written in another language must be able to read it.
	Payload     []byte
	TraceParent string
	Headers     map[string]string
}

// Message is a persisted message.
type Message struct {
	ID          MessageID
	Type        string
	AggregateID string
	Payload     []byte
	TraceParent string
	Headers     map[string]string
	Status      Status
	Attempts    int
	CreatedAt   time.Time
	AvailableAt time.Time
}

// FailedAttempt describes a publication failure to be recorded.
type FailedAttempt struct {
	ID          MessageID
	Attempts    int
	Status      Status
	AvailableAt time.Time
	Reason      string
}

// RetryPolicy bounds the dispatcher's persistence.
//
// Grouped into a type rather than separate parameters: the two values only make
// sense together — a number of attempts without a backoff, or the reverse,
// describes no policy at all. The grouping also makes it impossible to swap two
// arguments that the compiler would not tell apart.
type RetryPolicy struct {
	// MaxAttempts is the number of tries before definitive abandonment.
	MaxAttempts int
	// BaseBackoff is the step of the exponential backoff.
	BaseBackoff time.Duration
}

// maxShift bounds the exponent of the backoff.
//
// Without a bound, `1 << 40` would produce a NEGATIVE duration after overflow,
// hence a message immediately replayed in a tight loop — the exact opposite of
// what an exponential backoff must produce.
const maxShift = 10

// NextAttempt computes the state of a message after a failure.
//
// A PURE function: no clock, no randomness. The current instant is a parameter,
// which makes the retry policy testable without waiting.
func NextAttempt(msg Message, policy RetryPolicy, now time.Time, reason string) FailedAttempt {
	attempts := msg.Attempts + 1

	status := StatusPending
	if attempts >= policy.MaxAttempts {
		status = StatusFailed
	}

	shift := min(attempts, maxShift)

	return FailedAttempt{
		ID:          msg.ID,
		Attempts:    attempts,
		Status:      status,
		AvailableAt: now.Add(policy.BaseBackoff * time.Duration(1<<shift)),
		Reason:      reason,
	}
}

// IsDue states whether a message must be processed now.
func (m Message) IsDue(now time.Time) bool {
	return m.Status == StatusPending && !m.AvailableAt.After(now)
}
