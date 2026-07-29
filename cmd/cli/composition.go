package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/resources"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/hashing"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/outboxpub"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"
)

// mounted carries what the CLI has assembled, and what is needed to release it.
//
// A type rather than three return values: the architecture rule allows two,
// and this is the SEVENTH time this repository pays the same lesson.
type mounted struct {
	cfg   config.Config
	users userregistration.Module
	conn  resources.Connections
}

// compose loads the configuration and mounts what the CLI needs.
//
// # The SAME configuration as the other binaries
//
// The catalogue before the configuration, exactly as in `cmd/server` and
// `cmd/worker` (ADR 014). An administration binary reading its own
// configuration would end up acting on a store other than the one it claims to
// administer — and the defect would only show when noticing the absence of an
// account that was nevertheless created.
//
// # The log goes to the ERROR output
//
// A CLI's standard output is data: a script reads it. Mixing log lines into it
// would break any caller that runs an `awk` or a `cut` over it.
func compose(ctx context.Context) (mounted, error) {
	catalog, err := moduleCatalog()
	if err != nil {
		return mounted{}, err
	}

	cfg, err := config.Load(catalog)
	if err != nil {
		return mounted{}, fmt.Errorf("configuration: %w", err)
	}

	conn, err := resources.Open(ctx, cfg, catalog)
	if err != nil {
		return mounted{}, fmt.Errorf("opening the connections: %w", err)
	}

	users, err := mountRegistration(cfg, conn)
	if err != nil {
		conn.Close()
		return mounted{}, err
	}

	warnVolatileStore(cfg)
	return mounted{cfg: cfg, users: users, conn: conn}, nil
}

// warnVolatileStore says what an account created here is really worth.
//
// # Why a warning and not a refusal
//
// The dispatcher, for its part, REFUSES a non-shared driver: it would dispatch
// an empty store and run doing nothing. The reasoning is the same here — an
// account written into a store that dies with the process does not exist — but
// the conclusion differs, and for a precise reason: `user_registration` has NO
// other driver. Refusing would make this command permanently unusable, which
// is not a guard, it is a removal.
//
// The warning goes to the ERROR output: standard output is data a script cuts
// up, and slipping a sentence in there would break the cutting.
//
// The day a shared driver exists, this case will have to become a refusal —
// as for the dispatcher.
func warnVolatileStore(cfg config.Config) {
	if cfg.Modules.DriverOf(userregistration.Name) != userregistration.DriverMemory {
		return
	}
	fmt.Fprintln(os.Stderr,
		"warning: `memory` driver — the account created DIES with this process "+
			"and will be visible to no other binary. No shared driver exists "+
			"yet for user_registration.")
}

// mountRegistration wires the registration module.
//
// The driver and the hashing parameters come from the configuration, as in
// `cmd/server`: an account created by the CLI must be in every respect the one
// the HTTP surface would have created, otherwise the demonstration is worth
// nothing.
func mountRegistration(cfg config.Config, conn resources.Connections) (userregistration.Module, error) {
	outboxMod, err := outbox.New(cfg.Modules[outbox.Name], outbox.Deps{Pool: conn.Pool, Now: time.Now})
	if err != nil {
		return userregistration.Module{}, fmt.Errorf("outbox module: %w", err)
	}

	hasher := security.NewHasher(security.Argon2Params{
		MemoryKiB:  cfg.Security.Argon2.MemoryKiB,
		Iterations: cfg.Security.Argon2.Iterations,
		Threads:    cfg.Security.Argon2.Threads,
	})

	users, err := userregistration.New(
		cfg.Modules.DriverOf(userregistration.Name), userregistration.Deps{
			HashPassword: hashing.New(hasher),
			PublishEvent: outboxpub.New(outboxMod.Enqueue),
			GenerateID:   newIdentifier,
			Now:          userregistration.SystemClock(),
		})
	if err != nil {
		return userregistration.Module{}, fmt.Errorf("user_registration module: %w", err)
	}
	return users, nil
}

// newIdentifier produces a time-ordered identifier.
//
// UUID v7 and not v4: the identifier becomes a primary key, and a random key
// scatters the inserts across the whole index.
func newIdentifier() domain.UserID {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.UserID(uuid.NewString())
	}
	return domain.UserID(id.String())
}

// moduleCatalog assembles what the configuration is allowed to name.
func moduleCatalog() (config.ModuleCatalog, error) {
	coreCatalog, err := core.Catalog()
	if err != nil {
		return nil, fmt.Errorf("core catalogue: %w", err)
	}
	businessCatalog, err := modules.Catalog()
	if err != nil {
		return nil, fmt.Errorf("business module catalogue: %w", err)
	}
	merged, err := config.MergeCatalogs(coreCatalog, businessCatalog)
	if err != nil {
		return nil, fmt.Errorf("merging the catalogues: %w", err)
	}
	return merged, nil
}
