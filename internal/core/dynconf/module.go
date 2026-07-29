// Package dynconf is the core module of configuration changeable at run time.
//
// Composition root of the module: the only place that knows the drivers.
package dynconf

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/domain"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/drivers/file"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/drivers/postgres"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf/ports"
)

// Name is the name of the module in config/modules.yaml.
const Name = "dynconf"

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
	driverFile     = "file"
	driverPostgres = "postgres"
)

// defaultTTL bounds the freshness of the cache of the postgres driver.
//
// Thirty seconds is the compromise: short enough that a flag switched off in an
// emergency is off everywhere quickly, long enough that an evaluation in a loop
// does not hammer the database.
// Option keys of the module, shared with the catalogue (ADR 014, #93).
const (
	OptionFlags    = "flags"
	OptionSettings = "settings"
	OptionTTL      = "ttl"
)

const defaultTTL = 30 * time.Second

// Module exposes the ports of dynamic configuration.
type Module struct {
	IsEnabled  ports.IsEnabled
	GetSetting ports.GetSetting
	Set        ports.Set
	Invalidate ports.Invalidate
}

// Deps carries the dependencies the drivers may claim.
//
// All of them may be nil: the `file` driver claims none.
type Deps struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
	Now    func() time.Time
}

// ErrDisabled signals a write on a disabled module.
var ErrDisabled = errors.New("dynconf module disabled in config/modules.yaml")

// ErrPoolRequired signals a driver that requires an absent database.
var ErrPoolRequired = errors.New("the postgres driver requires a database connection")

// ErrLoggerRequired signals a driver that requires an absent logger.
//
// The postgres driver can NOT return its outages: the contract of
// ports.IsEnabled forbids it. It must therefore be able to log them, otherwise
// an unreachable database would switch off the hidden features without leaving
// a trace.
var ErrLoggerRequired = errors.New("the postgres driver requires a logger")

var errUnknownDriver = errors.New("unknown dynconf driver")

// New builds the module according to the configuration.
func New(cfg config.Module, deps Deps) (Module, error) {
	if !cfg.Enabled {
		return disabled(), nil
	}

	switch cfg.Driver {
	case driverFile:
		return fromFile(cfg)
	case driverPostgres:
		return fromPostgres(cfg, deps)
	default:
		return Module{}, fmt.Errorf("%w: %q", errUnknownDriver, cfg.Driver)
	}
}

// fromFile builds the driver of the versioned values.
func fromFile(cfg config.Module) (Module, error) {
	flags, err := cfg.MapOption(OptionFlags)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	settings, err := cfg.MapOption(OptionSettings)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	store, err := file.New(flags, settings)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	return Module{
		IsEnabled:  store.Flag,
		GetSetting: store.Setting,
		Set:        store.Set,
		Invalidate: store.Invalidate,
	}, nil
}

// fromPostgres builds the driver changeable at run time.
func fromPostgres(cfg config.Module, deps Deps) (Module, error) {
	if deps.Pool == nil {
		return Module{}, ErrPoolRequired
	}
	if deps.Logger == nil {
		return Module{}, ErrLoggerRequired
	}
	ttl, err := cfg.DurationOption(OptionTTL, defaultTTL)
	if err != nil {
		return Module{}, fmt.Errorf("modules.%s.%w", Name, err)
	}
	store := postgres.New(deps.Pool, deps.Logger, ttl, deps.Now)
	return Module{
		IsEnabled:  store.Flag,
		GetSetting: store.Setting,
		Set:        store.Set,
		Invalidate: store.Invalidate,
	}, nil
}

// disabled returns ports that are inert on READ and refusing on WRITE.
//
// # Why reading does not refuse here, unlike in the other modules
//
// `ports.IsEnabled` cannot return an error, by contract. The only possible
// answer is therefore `false` — and that is precisely the "deny by default"
// answer, the one that switches the hidden features off rather than on.
// Inertia coincides with refusal here, which is not the case elsewhere.
//
// `Set`, for its part, CAN speak: it refuses loudly. A caller that believed it
// had changed a flag on a switched-off module is the only real trap of this
// module.
func disabled() Module {
	return Module{
		IsEnabled:  func(context.Context, domain.FlagKey) bool { return false },
		GetSetting: func(context.Context, domain.SettingKey) domain.Setting { return domain.Setting{} },
		Set:        func(context.Context, domain.Change) error { return ErrDisabled },
		Invalidate: func() {},
	}
}
