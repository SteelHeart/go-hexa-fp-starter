// Command worker dispatches the outbox and publishes to the messaging relay.
//
// # The asynchronous half of the chain
//
// A use case writes its event into the outbox, INSIDE its business
// transaction: that is what guarantees no event is lost nor phantom (ADR 006).
// Without this binary, the event is durable and never published — the chain
// stops halfway, and nothing signals it on the server side.
//
// # This binary REFUSES to start on the `memory` driver, and that is intended
//
// The outbox's `memory` driver lives in the process. A separate worker would
// therefore dispatch ITS store, empty, while the server's events would stay in
// the server's memory. It would run publishing nothing, with no error at all,
// and the defect would only show at the consumer that would never have
// received anything.
//
// A silently inert component is the worst possible defect. The worker
// therefore refuses explicitly, saying what to change.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox"
	outboxapp "github.com/SteelHeart/go-hexa-fp-starter/internal/core/outbox/application"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/messaging"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/relay"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/resources"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/telemetry"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/modules"
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
		fmt.Fprintf(os.Stderr, "cannot start: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
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

	shutdown, err := startObservability(ctx, cfg)
	if err != nil {
		return err
	}
	defer stopObservability(ctx, shutdown, logger)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	serveMetrics(ctx, cfg, logger, cancel)

	logger.InfoContext(ctx, "dispatcher starting",
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("build_date", buildDate),
	)

	return runDispatcher(ctx, cfg, catalog, logger)
}

// runDispatcher opens the connections, mounts the dispatcher and holds it
// until shutdown.
//
// Extracted from `run` because that one went past the `arch-go` line threshold
// once the pool opening was wired into it. The guard caught the regression
// before review — which is exactly what it is asked to do, and it is the
// second time.
//
// The cut is not arbitrary: `run` carries the BOOTSTRAP — signals, catalogue,
// configuration, log, observability — and `runDispatcher` carries the WORK.
func runDispatcher(
	ctx context.Context, cfg config.Config, catalog config.ModuleCatalog, logger *slog.Logger,
) error {
	// The connections BEFORE the modules: a `postgres` driver receives a pool
	// or refuses to start, it never breaks down on the first request.
	//
	// Nothing is opened if no enabled module asks for it — that is the promise
	// of ADR 012, and fixing #103 was not to cost it.
	conn, err := resources.Open(ctx, cfg, catalog)
	if err != nil {
		return fmt.Errorf("opening the connections: %w", err)
	}
	defer conn.Close()

	w, err := compose(cfg, conn, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := w.closeBroker(); err != nil {
			logger.Error("closing the relay failed", slog.Any("error", err))
		}
	}()

	logger.InfoContext(ctx, "dispatching in progress",
		slog.String("outbox_driver", cfg.Modules[outbox.Name].Driver),
		slog.String("relais", cfg.Messaging.Driver),
	)

	if err := w.loop(ctx); err != nil {
		return err
	}
	logger.Info("dispatcher stopped gracefully")
	return nil
}

// worker is the mounted dispatcher, with what is needed to release it.
//
// A type rather than three return values: `compose` used to return
// `(*Dispatcher, Closer, error)`, and the architecture rule refused it. It is
// right, and this is the FIFTH time this repository pays the same lesson —
// after `election`, `decodedHash`, `RetryPolicy` and `messaging.Broker`. More
// than two return values always signals a missing type.
type worker struct {
	dispatch    *outboxapp.Dispatcher
	consume     messaging.Consumer
	closeBroker messaging.Closer
}

// compose mounts the relay, the outbox and the dispatcher.
//
// It returns the broker's releaser rather than closing it itself: the broker's
// lifetime is that of the process, not that of this function.
func compose(cfg config.Config, conn resources.Connections, logger *slog.Logger) (worker, error) {
	outboxCfg := cfg.Modules[outbox.Name]
	if !outboxCfg.Enabled {
		return worker{}, fmt.Errorf("%w: modules.outbox.enabled is false", outbox.ErrDisabled)
	}
	// The server has no such requirement: it WRITES into the outbox and
	// therefore has no reason to demand a shared driver. Only a SEPARATE
	// dispatcher needs one, and that is why the refusal lives here and not in
	// the module.
	if err := outbox.RequireSharedDriver(outboxCfg.Driver); err != nil {
		return worker{}, fmt.Errorf("dispatcher configuration: %w", err)
	}

	broker, err := messaging.New(cfg.Messaging, logger)
	if err != nil {
		return worker{}, fmt.Errorf("messaging relay: %w", err)
	}

	// The subscriptions BEFORE the dispatcher: `Subscribe` must be called
	// before `Run`, and a dispatcher publishing while the subscriptions are
	// being mounted would lose the first envelopes without saying so.
	//
	// A NAMED variable rather than a reused `err`: `govet shadow` and
	// `gocritic sloppyReassign` contradict each other on this line — one wants
	// `=`, the other `:=`. A distinct name steps out of the conflict without
	// silencing either, and reads better.
	if subscriptionErr := subscribe(cfg, broker, logger); subscriptionErr != nil {
		return worker{}, subscriptionErr
	}

	// The pool may legitimately be nil: it is as soon as no enabled module
	// asks for a database. A driver that needs one then refuses to start,
	// saying so — that is the guard which already existed and that nobody
	// reached.
	mod, err := outbox.New(outboxCfg, outbox.Deps{Pool: conn.Pool, Now: time.Now})
	if err != nil {
		return worker{}, fmt.Errorf("outbox module: %w", err)
	}

	dispatch, err := outbox.NewDispatcher(mod, outboxCfg, outbox.DispatcherDeps{
		Handle: relay.FromOutbox(broker.Publish),
		Logger: logger,
		Now:    time.Now,
	})
	if err != nil {
		return worker{}, fmt.Errorf("outbox dispatcher: %w", err)
	}
	return worker{dispatch: dispatch, consume: broker.Consume, closeBroker: broker.Close}, nil
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
