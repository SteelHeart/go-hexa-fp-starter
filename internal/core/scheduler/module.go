// Package scheduler is the core module of periodic tasks.
//
// Composition root of the module: the only place that knows the drivers.
//
// # Why the election is a port and the loop an orchestration
//
// The previous version of this building block merged the timer, the election by
// advisory lock and the logging into a single type. Consequence: it required
// PostgreSQL in order to repeat a task, including in a single-instance binary
// which has nobody to agree with. That is exactly what ADR 012 forbids.
//
// Separated, the loop can be tested without a database and the default driver
// no longer has any dependency.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/drivers/inproc"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/ports"
)

// Name is the name of the module in config/modules.yaml.
const Name = "scheduler"

// Names of the drivers of this module.
//
// They exist so that `Catalog` and the `switch` of `New` share the SAME
// identifier. This is what makes divergence between the two IMPOSSIBLE, where
// ADR 014 only promised to make it improbable — the compiler refuses a
// constant that does not exist, a misspelt literal goes through.
//
// The `goconst` linter reported the repetition as soon as the catalogue
// arrived. It was right, and for a stronger reason than its own.
const (
	driverCronInproc   = "cron-inproc"
	driverAdvisoryLock = "advisory-lock"
)

// Run launches the scheduler until the context is cancelled.
//
// Declared here and not in `ports/` because it names `application.Scheduled`:
// the ports of a core module depend only on their domain (arch-go.yml §4e).
type Run = func(ctx context.Context, scheduled []application.Scheduled) error

// Module exposes the ports of the scheduler.
type Module struct {
	// Run runs the tasks. This is the normal entry point of the module.
	Run Run
	// Acquire and Release are exposed for an EXTERNAL trigger — the scheduled
	// task of an orchestrator, the system timer — that wants the same election
	// without the loop.
	Acquire ports.Acquire
	Release ports.Release
}

// Deps carries the dependencies the drivers may claim.
type Deps struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
	Now    func() time.Time
}

// ErrDisabled signals a call to a disabled module.
var ErrDisabled = errors.New("scheduler module disabled in config/modules.yaml")

// ErrPoolRequired signals a driver that requires an absent database.
var ErrPoolRequired = errors.New("the advisory-lock driver requires a database connection")

// ErrLoggerRequired signals the absence of a logger.
//
// The orchestration does not log: it reports (ports.Report). The report must
// end up somewhere, otherwise a failing task would fail in silence — the worst
// possible defect for work that runs with no witness.
var ErrLoggerRequired = errors.New("the scheduler module requires a logger to report to")

var errUnknownDriver = errors.New("unknown scheduler driver")

// New builds the module according to the configuration.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}
	if deps.Logger == nil {
		return Module{}, ErrLoggerRequired
	}

	elected, err := elector(cfg, deps)
	if err != nil {
		return Module{}, err
	}

	now := deps.Now
	if now == nil {
		now = time.Now
	}
	runner, err := application.NewRunner(application.Ports{
		Acquire: elected.acquire,
		Release: elected.release,
		Report:  LogReport(deps.Logger),
		Now:     now,
	})
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s: %w", Name, err)
	}
	return Module{Run: runner.Run, Acquire: elected.acquire, Release: elected.release}, nil
}

// election carries the inseparable acquisition / release pair.
//
// Grouped into a type rather than into two return values: the two functions
// make no sense apart — acquiring without being able to release freezes a task
// for ever. The grouping also makes it impossible to swap two returns that the
// compiler would not tell apart.
type election struct {
	acquire ports.Acquire
	release ports.Release
}

// elector chooses the election mechanism.
func elector(cfg config.Module, deps Deps) (election, error) {
	switch cfg.Driver {
	case driverCronInproc:
		local := inproc.New()
		return election{acquire: local.Acquire, release: local.Release}, nil
	case driverAdvisoryLock:
		if deps.Pool == nil {
			return election{}, ErrPoolRequired
		}
		shared := postgres.New(deps.Pool)
		return election{acquire: shared.Acquire, release: shared.Release}, nil
	default:
		return election{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// disabled returns ports that refuse explicitly.
//
// Acquire returns `false` AND an error: `false` guarantees that no task runs,
// the error guarantees that the refusal is visible.
func disabled() Module {
	return Module{
		Run:     func(context.Context, []application.Scheduled) error { return ErrDisabled },
		Acquire: func(context.Context, domain.TaskName) (bool, error) { return false, ErrDisabled },
		Release: func(context.Context, domain.TaskName) error { return ErrDisabled },
	}
}
