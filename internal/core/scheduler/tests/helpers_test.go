// Package tests holds the BLACK BOX tests of the scheduler module: they only
// use the public API, exactly like a caller would.
//
// Repository convention (rules/tests.md): `{package}/tests/` for black box,
// `{package}/internal_test.go` for unexported identifiers. One file per test —
// the file name says what is verified, without having to open it.
package tests

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/ports"
)

// discardLogger satisfies the logger requirement without polluting the output
// of the tests.
//
// The module refuses a nil logger: a report that ends up nowhere would let a
// task fail in silence.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newInprocModule builds the module on its default driver.
func newInprocModule(t *testing.T) scheduler.Module {
	t.Helper()
	mod, err := scheduler.New(
		config.Module{Enabled: true, Driver: "cron-inproc"},
		scheduler.Deps{Logger: discardLogger()},
	)
	if err != nil {
		t.Fatalf("building the module: %v", err)
	}
	return mod
}

// startedAt is the reference instant. No test reads the real clock.
func startedAt() time.Time { return time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC) }

// clock is a driven clock: every read advances by a fixed step, which makes the
// measured durations deterministic without ever waiting.
type clock struct {
	mu   sync.Mutex
	at   time.Time
	step time.Duration
}

func newClock(step time.Duration) *clock {
	return &clock{at: startedAt(), step: step}
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.at
	c.at = c.at.Add(c.step)
	return current
}

// reportLog collects the reports. This is what the Report port makes possible:
// verifying an execution policy by reading values, not JSON.
type reportLog struct {
	mu       sync.Mutex
	outcomes []domain.Outcome
}

func (j *reportLog) report() ports.Report {
	return func(_ context.Context, outcome domain.Outcome) {
		j.mu.Lock()
		defer j.mu.Unlock()
		j.outcomes = append(j.outcomes, outcome)
	}
}

func (j *reportLog) all() []domain.Outcome {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]domain.Outcome(nil), j.outcomes...)
}

// count counts the reports of a given event.
func (j *reportLog) count(event domain.Event) int {
	total := 0
	for _, outcome := range j.all() {
		if outcome.Event == event {
			total++
		}
	}
	return total
}

// last returns the final report, or false if there is none.
func (j *reportLog) last() (domain.Outcome, bool) {
	all := j.all()
	if len(all) == 0 {
		return domain.Outcome{}, false
	}
	return all[len(all)-1], true
}

// alwaysElected always grants the execution.
func alwaysElected() (acquire ports.Acquire, release ports.Release) {
	return func(context.Context, domain.TaskName) (bool, error) { return true, nil },
		func(context.Context, domain.TaskName) error { return nil }
}

// newRunner builds a runner on closures.
func newRunner(t *testing.T, acquire ports.Acquire, release ports.Release, j *reportLog) *application.Runner {
	t.Helper()
	runner, err := application.NewRunner(application.Ports{
		Acquire: acquire,
		Release: release,
		Report:  j.report(),
		Now:     newClock(250 * time.Millisecond).now,
	})
	if err != nil {
		t.Fatalf("building the runner: %v", err)
	}
	return runner
}

// task describes a valid task.
func task(name string) domain.Task {
	return domain.Task{Name: domain.TaskName(name), Every: time.Hour, Timeout: time.Minute}
}
