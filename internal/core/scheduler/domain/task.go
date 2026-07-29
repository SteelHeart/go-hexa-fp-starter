// Package domain carries the vocabulary of periodic tasks, with no dependency.
//
// # The real problem is not periodicity
//
// A ticker is enough to repeat. The problem is that N replicas would run the
// task N times: a reminder sent three times, an invoice issued three times. All
// the value of the module is in the ELECTION, not in the clock — which is why
// the election is a port and the loop a simple orchestration.
package domain

import (
	"errors"
	"fmt"
	"hash/fnv"
	"time"
)

// TaskName identifies a task. It is also the election key: two tasks with the
// same name exclude each other, including between replicas.
type TaskName string

// String returns the name in raw form.
func (n TaskName) String() string { return string(n) }

// ErrInvalidTask refuses an unusable task.
var ErrInvalidTask = errors.New("invalid scheduled task")

// Task describes a periodic task. Deliberately without behaviour: the work is a
// port (ports.Job), not a struct field, so that the domain stays pure and the
// description of a task can be compared and logged.
type Task struct {
	Name  TaskName
	Every time.Duration
	// Timeout bounds one execution. At zero, the period serves as the bound: a
	// task that overruns its own period is already late, letting it run any
	// longer only piles executions up.
	Timeout time.Duration
}

// Validate refuses a task that cannot be run soundly.
//
// A null or negative period would make time.NewTicker panic: refusing at
// startup is better than a crash in the first minute of execution.
func (t Task) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("%w: name missing", ErrInvalidTask)
	}
	if t.Every <= 0 {
		return fmt.Errorf("%w: %s has a period of %v", ErrInvalidTask, t.Name, t.Every)
	}
	if t.Timeout < 0 {
		return fmt.Errorf("%w: %s has a negative timeout", ErrInvalidTask, t.Name)
	}
	return nil
}

// Deadline returns the bound of one execution.
func (t Task) Deadline() time.Duration {
	if t.Timeout > 0 {
		return t.Timeout
	}
	return t.Every
}

// Event names what happened to a task.
//
// Declared in the domain because the orchestration does NOT log: it reports
// (ports.Report). That is what keeps it pure — no logger, no I/O — and what
// allows a test to verify an execution policy by reading values rather than by
// parsing JSON.
type Event string

const (
	// EventSkipped: another replica holds the task. NOMINAL case, not an incident.
	EventSkipped Event = "skipped"
	// EventSucceeded: the work finished without error.
	EventSucceeded Event = "succeeded"
	// EventFailed: the work returned an error.
	EventFailed Event = "failed"
	// EventElectionFailed: the election mechanism is down. The task was NOT run
	// — that is the safe fallback.
	EventElectionFailed Event = "election_failed"
	// EventReleaseFailed: the work is done but the lock was not given back. The
	// task will not run again until the lock expires: to be watched.
	EventReleaseFailed Event = "release_failed"
)

// Outcome is the report of an execution attempt.
type Outcome struct {
	Task     TaskName
	Event    Event
	Duration time.Duration
	// Err carries the cause for EventFailed, EventElectionFailed and
	// EventReleaseFailed. Nil everywhere else.
	Err error
}

// LockKey derives a stable integer from the name of the task.
//
// Necessary because a PostgreSQL advisory lock is expressed as a `bigint`, not
// as text. The one-bit shift keeps the result positive: a negative key is valid
// on the database side but unreadable in a log and in `pg_locks`.
//
// # On the risk of collision
//
// A collision would make two DISTINCT tasks exclude each other. That is
// improbable over 63 bits, and above all the worst case is a serialised
// execution — never a double execution. The fallback therefore goes in the
// right direction.
func LockKey(name TaskName) int64 {
	digest := fnv.New64a()
	// fnv.Write never returns an error: the signature comes from io.Writer.
	_, _ = digest.Write([]byte("scheduler:" + name.String()))
	return int64(digest.Sum64() >> 1)
}
