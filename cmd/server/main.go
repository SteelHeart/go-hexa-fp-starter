// Command server exposes the starter's HTTP surfaces.
//
// # This is the composition root: the ONLY code allowed to know everything
//
// It knows the configuration, the drivers, the adapters and the modules. All
// the rest of the repository knows nothing but function types. That accepted
// imbalance is what keeps the core pure: some place has to wire the threads
// together, and that place is here, visible, rather than scattered across a
// dependency injection container (ADR 004).
//
// # Zero infrastructure prerequisite
//
// With the shipped configuration, this binary starts with no database, no
// Redis, no Docker: every default driver is in memory. That is the promise of
// ADR 012, and this file is where it is checked — `go run ./cmd/server` must
// work on a bare machine.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/resources"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/security"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/telemetry"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules"
	userregistration "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/hashing"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/secondary/outboxpub"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/domain"

	authhttp "github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth/adapters/primary/http"
	userhttp "github.com/SteelHeart/go-hexa-fp-starter/internal/modules/user_registration/adapters/primary/http"
)

// Injected at build time by the CI (see Dockerfile).
//
// Globals accepted: `-ldflags -X` can only write into a package variable.
// There is no other way to burn the version into the binary.
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		// Written to stderr and not through the logger: the failure may
		// precede the construction of the logger itself.
		fmt.Fprintf(os.Stderr, "cannot start: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// NotifyContext before anything else: a Ctrl+C during initialisation must
	// interrupt, not be swallowed.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The catalogue BEFORE the configuration: it is what says what the
	// configuration is allowed to name (ADR 014). No module table lives in
	// `internal/config` — it would name modules there, which `arch-go` rule 7
	// forbids it.
	catalog, err := moduleCatalog()
	if err != nil {
		return fmt.Errorf("module catalogue: %w", err)
	}

	cfg, err := config.Load(catalog)
	if err != nil {
		return fmt.Errorf("configuration: %w", err)
	}

	logger := telemetry.NewLogger(cfg)

	// Observability BEFORE everything else: an initialisation defect arising
	// after the modules are mounted would be traced nowhere.
	shutdown, err := startObservability(ctx, cfg)
	if err != nil {
		return err
	}
	defer stopObservability(ctx, shutdown, logger)

	// A derived context, so that the metrics server can cut the service off:
	// without it, its failure would stay a warning in a log.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	serveMetrics(ctx, cfg, logger, cancel)

	logger.InfoContext(ctx, "starting",
		slog.String("service", cfg.App.Name),
		slog.String("env", string(cfg.App.Env)),
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("build_date", buildDate),
	)

	// The connections BEFORE the surfaces: a `postgres` driver receives a pool
	// or refuses to start, it never breaks down on the first request.
	//
	// Nothing is opened if no enabled module asks for it — that is the promise
	// of ADR 012, and fixing #103 was not to cost it.
	conn, err := resources.Open(ctx, cfg, catalog)
	if err != nil {
		return fmt.Errorf("opening the connections: %w", err)
	}
	defer conn.Close()

	return serve(ctx, cfg, conn, logger)
}

// serve mounts the surfaces and holds the server until cancellation.
//
// Extracted from `run` because that one went past the `arch-go` line threshold
// once observability was wired into it. The guard caught the regression before
// review — which is exactly what it is asked to do.
//
// The cut is not arbitrary: `run` now carries the BOOTSTRAP — signals,
// catalogue, configuration, log, observability — and `serve` carries the
// SURFACES. Two responsibilities, two functions.
func serve(ctx context.Context, cfg config.Config, conn resources.Connections, logger *slog.Logger) error {
	mounted, err := compose(cfg, conn, logger)
	if err != nil {
		return err
	}

	// The bootstrap BEFORE the routes are mounted: an account announced on a
	// server that is not listening yet cannot be used in the meantime.
	if err := bootstrapAuthentication(ctx, mounted.auth, cfg.App.Env, logger); err != nil {
		return err
	}

	router := httpserver.NewRouter(cfg, logger, probes(mounted))
	authhttp.Mount(router.API, mounted.auth)
	userhttp.Mount(router.API, mounted.users)
	userhttp.MountAvailability(router.API, mounted.users)

	// The paths announced are the ones that ANSWER: huma serves the contract
	// under `/openapi.json` and `/openapi.yaml`, never under a bare `/openapi`
	// — which returns 404. Announcing a path that does not answer sends people
	// hunting for a failure that does not exist.
	logger.InfoContext(ctx, "surfaces mounted",
		slog.String("docs", "/docs"),
		slog.String("openapi", "/openapi.json · /openapi.yaml"),
	)

	if err := httpserver.New(cfg, router.Mux, logger).Run(ctx); err != nil {
		return fmt.Errorf("HTTP server: %w", err)
	}
	return nil
}

// assembled gathers the mounted modules.
//
// A named struct rather than multiple return values: at the third module a
// function would return four values, and "more than three returns = a missing
// type" is a lesson already paid for three times in this repository.
type assembled struct {
	outbox outbox.Module
	auth   auth.Module
	users  userregistration.Module
}

// compose wires the core modules then the business modules.
//
// The order is not arbitrary: a business module consumes the core's ports,
// never the other way round (ADR 012).
func compose(cfg config.Config, conn resources.Connections, logger *slog.Logger) (assembled, error) {
	// The pool may legitimately be nil: it is as soon as no enabled module
	// asks for a database, which is the case of the shipped configuration. A
	// `postgres` driver enabled without a database then REFUSES to start
	// rather than breaking down on the first request.
	//
	// ⚠️ This comment used to say "the pool is nil", in the present tense and
	// without any condition. It was accurate, and that was the defect: NO
	// binary opened a connection, so no `postgres` driver of the repository
	// was reachable (#103).
	outboxMod, err := outbox.New(cfg.Modules[outbox.Name], outbox.Deps{Pool: conn.Pool, Now: time.Now})
	if err != nil {
		return assembled{}, fmt.Errorf("outbox module: %w", err)
	}

	hasher := security.NewHasher(security.Argon2Params{
		MemoryKiB:  cfg.Security.Argon2.MemoryKiB,
		Iterations: cfg.Security.Argon2.Iterations,
		Threads:    cfg.Security.Argon2.Threads,
	})

	authMod, err := mountAuth(cfg, hasher)
	if err != nil {
		return assembled{}, err
	}

	// The driver comes from the configuration, which carries it at last.
	//
	// This field ALWAYS stayed empty before ADR 014: `config/modules.yaml`
	// only validated core modules, so declaring `user_registration` there made
	// the start-up refuse. The module silently fell back to its default, and
	// the reference slice bypassed the very mechanism it is meant to
	// demonstrate.
	driver := cfg.Modules.DriverOf(userregistration.Name)

	users, err := userregistration.New(driver, userregistration.Deps{
		HashPassword: hashing.New(hasher),
		PublishEvent: outboxpub.New(outboxMod.Enqueue),
		GenerateID:   generateUserID,
		Now:          userregistration.SystemClock(),
	})
	if err != nil {
		return assembled{}, fmt.Errorf("user_registration module: %w", err)
	}

	logger.Info("modules mounted",
		slog.String("outbox", cfg.Modules[outbox.Name].Driver),
		// The driver OR "disabled": announcing `auth=memory` for a module that
		// is off would send people looking for an authentication defect on the
		// store side, when the surface returns 503 because nobody enabled it.
		slog.String("auth", driverOrDisabled(cfg.Modules[auth.Name])),
		slog.String("user_registration", orDefault(driver, userregistration.DriverMemory)),
	)
	return assembled{outbox: outboxMod, auth: authMod, users: users}, nil
}

// mountAuth wires authentication onto the application's hashing.
//
// # Why the hashing comes from HERE and not from the module
//
// Argon2id is costly and parameterised, and its tuning belongs to the
// application's security configuration. A module choosing its own parameters
// would freeze them for everyone — and the starter has precisely one place
// where that decision is taken.
//
// Extracted from `compose` because that one went past the `arch-go` line
// threshold once the pool opening was wired into it. The guard caught the
// regression before review, for the second time in this file.
func mountAuth(cfg config.Config, hasher security.Hasher) (auth.Module, error) {
	mod, err := auth.New(cfg.Modules[auth.Name], auth.Deps{
		HashSecret:   hasher.Hash,
		VerifySecret: hasher.Verify,
		Now:          time.Now,
	})
	if err != nil {
		return auth.Module{}, fmt.Errorf("auth module: %w", err)
	}
	return mod, nil
}

// generateUserID produces a time-ordered identifier.
//
// UUID v7 and not v4: the identifier becomes a primary key, and a random key
// scatters the inserts across the whole index
// (rules/donnees-et-migrations.md §7).
//
// The fallback to v4 on failure is deliberate: `NewV7` only fails when the
// system entropy is unavailable, and refusing a registration for that reason
// would be worse than a less well ordered identifier.
func generateUserID() domain.UserID {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.UserID(uuid.NewString())
	}
	return domain.UserID(id.String())
}

// probes declares what /readyz checks.
//
// /healthz checks NOTHING, deliberately: otherwise a database incident would
// restart every container, turning a partial failure into total
// unavailability. /readyz, on the other hand, removes the instance from the
// service.
func probes(mods assembled) map[string]httpserver.Probe {
	return map[string]httpserver.Probe{
		// The outbox is the dependency whose failure is SILENT: if the
		// dispatcher dies, everything keeps answering while the events pile
		// up. Counting the pending messages is the only observable symptom,
		// hence the most useful probe of the system.
		"outbox": func(ctx context.Context) error {
			if _, err := mods.outbox.PendingCount(ctx); err != nil {
				return fmt.Errorf("outbox unreachable: %w", err)
			}
			return nil
		},
	}
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// driverOrDisabled returns a module's driver, or its inactive state.
//
// A log that names a driver for a module that is off is a log that lies
// usefully: it answers the question nobody asked, and hides the one that
// matters. This is measured — a production start-up announced `auth=memory`
// while the surface was returning 503.
func driverOrDisabled(mod config.Module) string {
	if !mod.Enabled {
		return "disabled"
	}
	return mod.Driver
}

// moduleCatalog assembles what the configuration is allowed to name.
//
// The core brings its own; this binary adds that of every business module it
// embeds. No framework file names a business module — that is very exactly
// the friction ADR 014 removes.
func moduleCatalog() (config.ModuleCatalog, error) {
	coreCatalog, err := core.Catalog()
	if err != nil {
		return nil, fmt.Errorf("core catalogue: %w", err)
	}
	// Both binaries read the SAME `config/modules.yaml`: the set of declarable
	// modules is therefore a property of the application, not of the binary.
	// See internal/modules/catalog.go — the dispatcher refused to start until
	// that was the case.
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
