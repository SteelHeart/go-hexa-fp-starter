// Package application dispatches the outbox: claim, publish, record the fate.
//
// This package knows NO driver and does NO I/O: it receives the store, the
// publisher, the report and the clock as function types. That is what makes it
// possible to prove the dispatching policy — exponential backoff, giving up
// after N attempts, surviving a poisoned message — with closures, without a
// database and without waiting.
//
// In accordance with `rules/README.md` § « the core is pure »: no `time.Now()`,
// no logger, no `panic` raised here.
//
// # Why this dispatcher does not use the scheduler module
//
// The scheduler exists to elect ONE replica out of N. The dispatcher does not
// need it: the contract of `ports.Claim` already guarantees that two concurrent
// calls never return the same message — `FOR UPDATE SKIP LOCKED` on the
// postgres side, a local lock on the memory side. Several dispatchers therefore
// work IN PARALLEL without coordinating, which is better than a single elected
// one: throughput grows with the replicas instead of being capped by the
// slowest.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// Ports carries the contracts the dispatcher needs.
//
// Grouped in a struct rather than as parameters: six positional arguments of
// the same shape get swapped at the first addition, and the compiler would say
// nothing — `MarkDone` and `MarkFailed` have the same signature but for one
// type.
type Ports struct {
	Claim      ports.Claim
	MarkDone   ports.MarkDone
	MarkFailed ports.MarkFailed
	Handle     ports.Handler
	Report     ports.Report
	Now        ports.Now
}

// Policy carries the dispatching policy.
type Policy struct {
	// BatchSize bounds a batch. Too large, and a replica monopolises the work and
	// holds its locks for a long time; too small, and throughput collapses into
	// round trips.
	BatchSize int
	// Interval is the polling period when the dispatcher runs alone.
	Interval time.Duration
	// Retry belongs to the DOMAIN: it is the one that knows what a bounded
	// exponential backoff is, and it is the one that is tested to prove it.
	// Orchestration merely carries it along.
	Retry domain.RetryPolicy
}

// ErrMissingPort refuses an incomplete dispatcher.
var ErrMissingPort = errors.New("missing port for the outbox dispatcher")

// ErrInvalidPolicy refuses an unusable policy.
var ErrInvalidPolicy = errors.New("invalid dispatching policy")

// ErrHandlerPanicked signals a publisher that panicked.
var ErrHandlerPanicked = errors.New("the publisher panicked")

// Dispatcher dispatches the outbox.
type Dispatcher struct {
	ports  Ports
	policy Policy
}

// NewDispatcher builds the dispatcher.
//
// Refuses a nil port or an absurd policy rather than panicking at the first
// round: `time.NewTicker(0)` panics, and a panic in a worker's goroutine takes
// down the whole process.
func NewDispatcher(p Ports, policy Policy) (*Dispatcher, error) {
	if p.Claim == nil || p.MarkDone == nil || p.MarkFailed == nil ||
		p.Handle == nil || p.Report == nil || p.Now == nil {
		return nil, ErrMissingPort
	}
	if err := validate(policy); err != nil {
		return nil, err
	}
	return &Dispatcher{ports: p, policy: policy}, nil
}

// validate refuses a policy that would produce absurd behaviour.
func validate(policy Policy) error {
	switch {
	case policy.BatchSize <= 0:
		return fmt.Errorf("%w: batch of %d message(s)", ErrInvalidPolicy, policy.BatchSize)
	case policy.Retry.MaxAttempts <= 0:
		// Zero attempts would mean « give up before trying »: every message would
		// go to `failed` without any publication being attempted.
		return fmt.Errorf("%w: %d allowed attempt(s)", ErrInvalidPolicy, policy.Retry.MaxAttempts)
	case policy.Retry.BaseBackoff <= 0:
		// A zero backoff would replay a failed message without any pause, in a loop.
		return fmt.Errorf("%w: base backoff of %v", ErrInvalidPolicy, policy.Retry.BaseBackoff)
	case policy.Interval <= 0:
		return fmt.Errorf("%w: polling period of %v", ErrInvalidPolicy, policy.Interval)
	}
	return nil
}

// Run dispatches until the context is cancelled.
func (d *Dispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.policy.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Cancellation is the NORMAL end of a worker: a clean shutdown must not
			// look like an outage, neither in the exit code nor in the alerts.
			if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("outbox dispatcher interrupted: %w", err)
			}
			return nil
		case <-ticker.C:
			d.catchUp(ctx)
		}
	}
}

// maxRounds bounds the catch-up of a polling round.
//
// Without a bound, a backlog of a hundred thousand messages would monopolise
// the loop: the `select` would never be reached again, and the worker would
// ignore the shutdown request until it had drained everything — a deployment
// would turn into a wait.
const maxRounds = 10

// catchUp chains batches as long as they come back full.
//
// A full batch means « there are probably more »: waiting for the next tick
// would advance the catch-up by a single batch per period, and the backlog
// would never be absorbed if the input rate exceeds one batch per round.
func (d *Dispatcher) catchUp(ctx context.Context) {
	for range maxRounds {
		if ctx.Err() != nil {
			return
		}
		count, err := d.DrainOnce(ctx)
		if err != nil || count < d.policy.BatchSize {
			return
		}
	}
}

// DrainOnce claims a batch and processes it. Returns the number of messages
// processed.
//
// Exported so that an external trigger — a scheduler, a test, an operations
// command — applies exactly the same policy without the loop.
func (d *Dispatcher) DrainOnce(ctx context.Context) (int, error) {
	messages, err := d.ports.Claim(ctx, d.policy.BatchSize)
	if err != nil {
		d.ports.Report(ctx, domain.Outcome{Event: domain.EventClaimFailed, Err: err})
		return 0, fmt.Errorf("claiming a batch from the outbox: %w", err)
	}

	for _, msg := range messages {
		d.deliver(ctx, msg)
	}
	return len(messages), nil
}

// deliver publishes a message and records its fate.
func (d *Dispatcher) deliver(ctx context.Context, msg domain.Message) {
	started := d.ports.Now()
	err := d.publish(ctx, msg)
	elapsed := d.ports.Now().Sub(started)

	if err != nil {
		d.reschedule(ctx, msg, elapsed, err)
		return
	}

	if markErr := d.ports.MarkDone(ctx, msg.ID); markErr != nil {
		// Published, but not marked: the message stays « pending » and will be
		// republished. The consumer will receive a duplicate — hence the
		// obligation of idempotency on the consumer side (ports.Handler).
		d.report(ctx, msg, domain.Outcome{
			Event: domain.EventResolveFailed, Attempts: msg.Attempts, Duration: elapsed, Err: markErr,
		})
		return
	}

	d.report(ctx, msg, domain.Outcome{
		Event: domain.EventPublished, Attempts: msg.Attempts, Duration: elapsed,
	})
}

// publish calls the publisher while shielding it from a panic.
//
// # Why a recover here, when the core never panics
//
// The rule forbids RAISING a panic in the core. The publisher, however, is
// caller code: a nil map access in a consumer is enough. Without this net, a
// single poisoned message would kill the worker — and, with the memory driver,
// would leave it claimed forever. Treated as an ordinary failure, it follows the
// backoff policy and then ends up in `failed`, which is exactly the right fate.
func (d *Dispatcher) publish(ctx context.Context, msg domain.Message) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w (%s): %v", ErrHandlerPanicked, msg.Type, recovered)
		}
	}()

	if handleErr := d.ports.Handle(ctx, msg); handleErr != nil {
		return fmt.Errorf("publishing %s (%s): %w", msg.ID, msg.Type, handleErr)
	}
	return nil
}

// reschedule applies the retry policy after a failure.
//
// Decides nothing by itself: the backoff computation and the decision to give
// up come from domain.NextAttempt, which is pure and tested separately.
func (d *Dispatcher) reschedule(ctx context.Context, msg domain.Message, elapsed time.Duration, cause error) {
	attempt := domain.NextAttempt(msg, d.policy.Retry, d.ports.Now(), cause.Error())

	event := domain.EventRetryScheduled
	switch {
	case errors.Is(cause, ErrHandlerPanicked):
		event = domain.EventHandlerPanicked
	case attempt.Status == domain.StatusFailed:
		event = domain.EventExhausted
	}

	if markErr := d.ports.MarkFailed(ctx, attempt); markErr != nil {
		d.report(ctx, msg, domain.Outcome{
			Event: domain.EventResolveFailed, Attempts: attempt.Attempts,
			Duration: elapsed, Err: markErr,
		})
		return
	}

	d.report(ctx, msg, domain.Outcome{
		Event: event, Attempts: attempt.Attempts, Duration: elapsed, Err: cause,
	})
}

// report completes a report with the message's identity.
func (d *Dispatcher) report(ctx context.Context, msg domain.Message, outcome domain.Outcome) {
	outcome.ID = msg.ID
	outcome.Type = msg.Type
	d.ports.Report(ctx, outcome)
}
