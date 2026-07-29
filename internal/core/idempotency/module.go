// Package idempotency is the core module that makes a write replayable.
//
// This is the module's composition root: the ONLY place that knows the drivers.
// A caller receives function types and ignores which one is wired in
// ([ADR 012](../../../documentation/adr/012-anatomie-d-un-module-et-pilotes.md)).
package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/memory"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/drivers/redis"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/idempotency/ports"
)

// Name is the module's name in config/modules.yaml.
const Name = "idempotency"

// Driver names for this module.
//
// They exist so that `Catalog` and the `switch` in `New` share the SAME
// identifier. That is what makes divergence between the two IMPOSSIBLE, where
// ADR 014 only promised to make it improbable — the compiler refuses a constant
// that does not exist, a misspelt literal goes through.
//
// The `goconst` linter flagged the repetition as soon as the catalogue arrived.
// It was right, and for a stronger reason than its own.
const (
	driverMemory   = "memory"
	driverPostgres = "postgres"
	driverRedis    = "redis"
)

// defaultTTL is the window during which a replay is recognised.
//
// Twenty-four hours covers the case that motivates the module: a mobile client
// offline that resumes the next day. Beyond that, a replay recreates the
// resource — that is a trade-off, not an oversight, and it is tuned through
// `options.ttl`.
const defaultTTL = 24 * time.Hour

// defaultNamespace prefixes the keys of the redis driver.
const defaultNamespace = "idempotency"

// Option keys of the module, shared with the catalogue (ADR 014, #93).
const (
	OptionTTL       = "ttl"
	OptionNamespace = "namespace"
)

// Module exposes the idempotency ports.
type Module struct {
	Reserve  ports.Reserve
	Complete ports.Complete
	Release  ports.Release
	Purge    ports.Purge
}

// Deps carries the dependencies the drivers may claim.
//
// Pool and Cache may be nil: the `memory` driver needs neither of the two, and
// that is precisely what allows starting without a database and without Redis.
type Deps struct {
	Pool  *pgxpool.Pool
	Cache *goredis.Client
	Now   func() time.Time
}

// ErrDisabled reports a call to a disabled module.
//
// A disabled module fails explicitly rather than letting things through: an
// inert idempotency that "works anyway" would allow duplicates without ever
// reporting itself. That is the worst possible defect, because it is invisible
// until the day it costs dearly.
var ErrDisabled = errors.New("idempotency module disabled in config/modules.yaml")

// ErrPoolRequired reports a driver that requires an absent database.
var ErrPoolRequired = errors.New("the postgres driver requires a database connection")

// ErrCacheRequired reports a driver that requires an absent cache.
var ErrCacheRequired = errors.New("the redis driver requires a cache connection")

var errUnknownDriver = errors.New("unknown idempotency driver")

// New builds the module according to the configuration.
//
// An unknown driver refuses startup: configuration validation has already
// rejected it, and this second refusal guarantees that no path bypasses the
// first one.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}

	ttl, err := cfg.DurationOption(OptionTTL, defaultTTL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}

	switch cfg.Driver {
	case driverMemory:
		return fromMemory(memory.New(ttl, deps.Now)), nil
	case driverPostgres:
		if deps.Pool == nil {
			return Module{}, ErrPoolRequired
		}
		return fromPostgres(postgres.New(deps.Pool, ttl)), nil
	case driverRedis:
		return withRedis(cfg, deps, ttl)
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// withRedis builds the redis driver, whose options are richer.
func withRedis(cfg config.Module, deps Deps, ttl time.Duration) (Module, error) {
	if deps.Cache == nil {
		return Module{}, ErrCacheRequired
	}
	namespace, err := cfg.StringOption(OptionNamespace, defaultNamespace)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	store := redis.New(deps.Cache, namespace, ttl)
	return Module{
		Reserve:  store.Reserve,
		Complete: store.Complete,
		Release:  store.Release,
		Purge:    store.Purge,
	}, nil
}

// fromMemory assembles the ports of the memory driver.
func fromMemory(store *memory.Store) Module {
	return Module{
		Reserve:  store.Reserve,
		Complete: store.Complete,
		Release:  store.Release,
		Purge:    store.Purge,
	}
}

// fromPostgres assembles the ports of the postgres driver.
func fromPostgres(store *postgres.Store) Module {
	return Module{
		Reserve:  store.Reserve,
		Complete: store.Complete,
		Release:  store.Release,
		Purge:    store.Purge,
	}
}

// disabled returns ports that refuse explicitly.
func disabled() Module {
	return Module{
		Reserve: func(context.Context, domain.Request) (domain.Reservation, error) {
			return domain.Reservation{}, ErrDisabled
		},
		Complete: func(context.Context, domain.Key, []byte) error { return ErrDisabled },
		Release:  func(context.Context, domain.Key) error { return ErrDisabled },
		Purge:    func(context.Context) (int64, error) { return 0, ErrDisabled },
	}
}
