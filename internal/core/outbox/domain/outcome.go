package domain

import "time"

// Event names what happened to a message during its dispatching.
//
// Declared in the domain because orchestration does NOT log: it reports
// (ports.Report). That is what keeps it pure — no logger, no I/O — and what
// allows a test to check a dispatching policy by reading values rather than by
// parsing JSON.
type Event string

const (
	// EventPublished: the message went out and its fate is recorded. The only
	// case where the chain is genuinely closed.
	EventPublished Event = "published"
	// EventRetryScheduled: publication failed, a new attempt is scheduled.
	EventRetryScheduled Event = "retry_scheduled"
	// EventExhausted: the attempts are exhausted. The message goes to `failed`
	// and is no longer replayed — it stays in the database as the ONLY trace of
	// what has not been published. Someone must look.
	EventExhausted Event = "exhausted"
	// EventClaimFailed: impossible to claim a batch. The store is down; nothing
	// has been published and nothing is lost.
	EventClaimFailed Event = "claim_failed"
	// EventResolveFailed is the most serious, and the least obvious.
	//
	// The message has been published, but its fate could NOT be recorded. It
	// therefore stays « pending » and will be republished on the next round: the
	// consumer will receive a duplicate. That is precisely why delivery is « at
	// least once » and why every consumer must be idempotent.
	EventResolveFailed Event = "resolve_failed"
	// EventHandlerPanicked: the publisher panicked. Treated as an ordinary
	// failure so that the dispatcher survives a poisoned message.
	EventHandlerPanicked Event = "handler_panicked"
)

// Outcome is the report of a message's processing.
type Outcome struct {
	ID   MessageID
	Type string
	// Event says what happened.
	Event Event
	// Attempts is the number of attempts AFTER this one.
	Attempts int
	// Duration measures the call to the publisher alone, not the recording of
	// the fate.
	Duration time.Duration
	// Err carries the cause for any event other than EventPublished.
	Err error
}
