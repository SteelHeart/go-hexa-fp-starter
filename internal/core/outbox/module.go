// Package outbox is the core module for guaranteed publication.
//
// It is the module's composition root: the ONLY place that knows the drivers.
// A caller receives function types and ignores which one is wired
// ([ADR 012](../../../documentation/adr/012-anatomie-d-un-module-et-pilotes.md)).
package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/memory"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/ports"
)

// Name is the module's name in config/modules.yaml.
const Name = "outbox"

// Outbox drivers.
//
// Named once so that the wiring and the sharing table
// (SharedAcrossProcesses) cannot speak of different drivers.
const (
	driverMemory   = "memory"
	driverPostgres = "postgres"
)

// Module exposes the outbox ports.
type Module struct {
	Enqueue      ports.Enqueue
	Claim        ports.Claim
	MarkDone     ports.MarkDone
	MarkFailed   ports.MarkFailed
	PendingCount ports.PendingCount
}

// Deps carries the dependencies the drivers may claim.
//
// Pool may be nil: the `memory` driver does not need one, and that is precisely
// what allows starting without a database.
type Deps struct {
	Pool *pgxpool.Pool
	Now  func() time.Time
}

// ErrDisabled signals a call to a disabled module.
//
// A disabled module fails explicitly rather than falling back on inert
// behaviour: a silently ignored event is the worst possible defect, because it
// never signals itself.
var ErrDisabled = errors.New("outbox module disabled in config/modules.yaml")

// ErrPoolRequired signals a driver that requires an absent database.
var ErrPoolRequired = errors.New("the postgres driver requires a database connection")

// New builds the module according to the configuration.
//
// An unknown driver refuses startup: configuration validation has already
// rejected it, and this second refusal guarantees that no path bypasses the
// first.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}

	switch cfg.Driver {
	case driverMemory:
		store := memory.New(deps.Now)
		return Module{
			Enqueue:      store.Enqueue,
			Claim:        store.Claim,
			MarkDone:     store.MarkDone,
			MarkFailed:   store.MarkFailed,
			PendingCount: store.PendingCount,
		}, nil

	case driverPostgres:
		if deps.Pool == nil {
			return Module{}, ErrPoolRequired
		}
		store := postgres.New(deps.Pool)
		return Module{
			Enqueue:      store.Enqueue,
			Claim:        store.Claim,
			MarkDone:     store.MarkDone,
			MarkFailed:   store.MarkFailed,
			PendingCount: store.PendingCount,
		}, nil

	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

var errUnknownDriver = errors.New("unknown outbox driver")

// disabled returns ports that refuse explicitly.
func disabled() Module {
	return Module{
		Enqueue: func(context.Context, domain.NewMessage) (domain.MessageID, error) {
			return "", ErrDisabled
		},
		Claim:        func(context.Context, int) ([]domain.Message, error) { return nil, ErrDisabled },
		MarkDone:     func(context.Context, domain.MessageID) error { return ErrDisabled },
		MarkFailed:   func(context.Context, domain.FailedAttempt) error { return ErrDisabled },
		PendingCount: func(context.Context) (int64, error) { return 0, ErrDisabled },
	}
}
