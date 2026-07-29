// Package audit is the core module of audit logging.
//
// Composition root of the module: the only place that knows the drivers.
package audit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/domain"
	logdriver "github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/drivers/log"
	pgdriver "github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit/ports"
)

// Name is the name of the module in config/modules.yaml.
const Name = "audit"

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
	driverLog      = "log"
	driverPostgres = "postgres"
)

// Module exposes the audit port.
type Module struct{ Record ports.Record }

// Deps carries the dependencies of the drivers.
//
// Pool may be nil: the `log` driver does not need one.
type Deps struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
	Now    func() time.Time
}

// Errors of the module.
var (
	ErrDisabled       = errors.New("audit module disabled in config/modules.yaml")
	ErrPoolRequired   = errors.New("the postgres driver requires a database connection")
	ErrLoggerRequired = errors.New("the log driver requires a logger")
	errUnknownDriver  = errors.New("unknown audit driver")
)

// New builds the module according to the configuration.
//
// An unknown driver refuses to start: configuration validation has already
// rejected it, and this second refusal guarantees that no path bypasses the
// first.
func New(cfg config.Module, deps Deps) (Module, error) {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if !cfg.Enabled {
		return disabled(), nil
	}

	switch cfg.Driver {
	case driverLog:
		return newLog(deps)
	case driverPostgres:
		return newPostgres(deps)
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// disabled returns a module that refuses when called.
//
// A disabled audit must not return `nil` silently: an audit trace one believes
// written and which is not is worse than no audit at all.
func disabled() Module {
	return Module{Record: func(context.Context, domain.Entry) error { return ErrDisabled }}
}

func newLog(deps Deps) (Module, error) {
	if deps.Logger == nil {
		return Module{}, ErrLoggerRequired
	}
	return Module{Record: logdriver.New(deps.Logger, deps.Now)}, nil
}

func newPostgres(deps Deps) (Module, error) {
	if deps.Pool == nil {
		return Module{}, ErrPoolRequired
	}
	return Module{Record: pgdriver.New(deps.Pool, deps.Now)}, nil
}
