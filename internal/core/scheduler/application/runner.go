// Package application orchestrates the execution of periodic tasks.
//
// This package knows NO driver and does NO I/O: it receives the election, the
// report and the clock in the form of function types. That is what makes it
// possible to test it with closures, without a database and without waiting —
// the execution policy is the part we really want to prove.
//
// In accordance with `rules/README.md` § "the core is pure": no `time.Now()`,
// no logger, no `panic` here. `time.NewTicker` stays, because a timer is
// neither a clock read nor an input-output — and because a scheduler without a
// timer schedules nothing.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/ports"
)

// Scheduled associates a task description with the work to be run.
type Scheduled struct {
	Task domain.Task
	Job  ports.Job
}

// Ports carries the contracts the orchestration needs.
//
// Grouped in a struct rather than passed as parameters: four positional
// arguments of the same shape — three functions and a clock — get swapped at
// the first addition, and the compiler would say nothing.
type Ports struct {
	Acquire ports.Acquire
	Release ports.Release
	Report  ports.Report
	Now     ports.Now
}

// Runner runs the registered tasks.
type Runner struct{ ports Ports }

// ErrMissingPort refuses an incomplete orchestration.
var ErrMissingPort = errors.New("missing port for the scheduler")

// ErrNoJob refuses a task with no associated work.
var ErrNoJob = errors.New("scheduled task without work")

// NewRunner builds the runner.
//
// Refuses a missing port rather than letting a nil panic on the first tick: a
// panic in a scheduler goroutine takes the whole process with it.
func NewRunner(p Ports) (*Runner, error) {
	if p.Acquire == nil || p.Release == nil || p.Report == nil || p.Now == nil {
		return nil, ErrMissingPort
	}
	return &Runner{ports: p}, nil
}

// Run runs every task until the context is cancelled.
//
// Validates EVERYTHING before starting anything: launching three tasks out of
// four then failing would leave a half-alive scheduler, the hardest state to
// diagnose.
func (r *Runner) Run(ctx context.Context, scheduled []Scheduled) error {
	if err := validate(scheduled); err != nil {
		return err
	}

	done := make(chan struct{}, len(scheduled))
	for _, entry := range scheduled {
		go func(s Scheduled) {
			defer func() { done <- struct{}{} }()
			r.loop(ctx, s)
		}(entry)
	}
	for range scheduled {
		<-done
	}

	// Cancellation is the NORMAL end of a scheduler: a clean stop must not look
	// like an outage, neither in the reports, nor in the return code.
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("scheduler interrupted: %w", err)
	}
	return nil
}

// validate refuses an unusable batch of tasks.
func validate(scheduled []Scheduled) error {
	seen := make(map[domain.TaskName]struct{}, len(scheduled))
	for _, entry := range scheduled {
		if err := entry.Task.Validate(); err != nil {
			return fmt.Errorf("scheduled task refused: %w", err)
		}
		if entry.Job == nil {
			return fmt.Errorf("%w: %s", ErrNoJob, entry.Task.Name)
		}
		// Two tasks with the same name share the election key: they would
		// exclude each other and one of the two would never run, without ever
		// signalling itself other than by "skipped".
		if _, duplicate := seen[entry.Task.Name]; duplicate {
			return fmt.Errorf("%w: %s declared twice", domain.ErrInvalidTask, entry.Task.Name)
		}
		seen[entry.Task.Name] = struct{}{}
	}
	return nil
}

// loop repeats a task until cancellation.
func (r *Runner) loop(ctx context.Context, s Scheduled) {
	ticker := time.NewTicker(s.Task.Every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.RunOnce(ctx, s)
		}
	}
}

// RunOnce attempts a single execution: election, work, release.
//
// Exported so that an EXTERNAL trigger — the system scheduler, the scheduled
// task of an orchestrator — applies exactly the same policy, and so that a test
// verifies it without depending on a timer.
func (r *Runner) RunOnce(ctx context.Context, s Scheduled) {
	runCtx, cancel := context.WithTimeout(ctx, s.Task.Deadline())
	defer cancel()

	elected, err := r.ports.Acquire(runCtx, s.Task.Name)
	if err != nil {
		r.ports.Report(runCtx, domain.Outcome{
			Task: s.Task.Name, Event: domain.EventElectionFailed, Err: err,
		})
		return
	}
	if !elected {
		r.ports.Report(runCtx, domain.Outcome{Task: s.Task.Name, Event: domain.EventSkipped})
		return
	}

	// The release uses a context WITHOUT cancellation: if the work has
	// exhausted the timeout, releasing with the same context would fail and the
	// lock would stay taken until it expired — the task would not run during
	// all that time.
	defer func() {
		if err := r.ports.Release(context.WithoutCancel(runCtx), s.Task.Name); err != nil {
			r.ports.Report(runCtx, domain.Outcome{
				Task: s.Task.Name, Event: domain.EventReleaseFailed, Err: err,
			})
		}
	}()

	r.execute(runCtx, s)
}

// execute launches the work and reports on its fate.
func (r *Runner) execute(ctx context.Context, s Scheduled) {
	started := r.ports.Now()
	err := s.Job(ctx)
	elapsed := r.ports.Now().Sub(started)

	event := domain.EventSucceeded
	if err != nil {
		event = domain.EventFailed
	}
	r.ports.Report(ctx, domain.Outcome{
		Task: s.Task.Name, Event: event, Duration: elapsed, Err: err,
	})
}
