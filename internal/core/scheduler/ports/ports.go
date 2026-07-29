// Package ports declares the contracts of periodic tasks.
//
// This package contains ONLY type declarations.
package ports

import (
	"context"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
)

// Acquire attempts to become the runner of a task.
//
// # Conformance contract
//
//   - `true` means "nobody else is running it, go ahead". The caller MUST call
//     Release afterwards, even if the work fails.
//   - `false` is NOT an error: it is the nominal case on every non-elected
//     replica, and it happens on every tick on N-1 replicas.
//   - An error signals an outage of the election mechanism. When in doubt, a
//     driver returns `false`: not running is benign, running twice is not — a
//     reminder sent twice is visible to the customer.
type Acquire = func(ctx context.Context, task domain.TaskName) (bool, error)

// Release hands the execution back to the next one.
//
// Must be called with a NON-cancelled context: releasing with an already dead
// context would leave the lock in place until it expires, and the task would
// not run during that time.
type Release = func(ctx context.Context, task domain.TaskName) error

// Report reports on an execution attempt.
//
// The orchestration does not log: it reports. That is what keeps it pure —
// `rules/README.md` forbids any logger in `application/` — and that is what
// allows a test to verify an execution policy by reading values.
//
// Returns nothing: a report that failed must never make the task it reports on
// fail.
type Report = func(ctx context.Context, outcome domain.Outcome)

// Now returns the current instant.
//
// The orchestration does not read the system clock: it receives this port.
// Without that, no measurement of duration would be verifiable in a test.
type Now = func() time.Time

// Job is the work to be run.
//
// Supplied by the caller, never by the module: the starter knows when to run,
// it has no idea what to run.
//
// A Job MUST be idempotent. Even with a correct election, a replica killed
// between its work and its release leaves a partial execution that another
// replica will pick up.
type Job = func(ctx context.Context) error
