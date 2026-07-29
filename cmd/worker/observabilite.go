package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/telemetry"
)

// Observability wiring of the dispatcher — #13.
//
// The counterpart of `cmd/server/observabilite.go`, and it is necessary for a
// reason of the dispatcher's own: it is the only process whose inactivity
// NOBODY notices. A mute server shows at the first request; a dispatcher that
// no longer dispatches only shows at the moment one looks for an event that
// never left.
//
// Two composition roots, two files: that is the accepted counterpart of
// ADR 004. The only place that knows everything is this one, and sharing it
// would amount to building a miniature injection container.

const telemetryGrace = 5 * time.Second

// startObservability installs the providers and returns the shutdown function.
//
// Disabled — the shipped default —, `Setup` returns an inert shutdown: the
// dispatcher therefore always starts with no collector.
func startObservability(ctx context.Context, cfg config.Config) (telemetry.Shutdown, error) {
	shutdown, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("telemetry: %w", err)
	}
	return shutdown, nil
}

// stopObservability flushes the exporters before handing back control.
//
// A context DETACHED from the cancellation: at shutdown time `ctx` is already
// cancelled, and passing it as is would drop the buffered spans — those of the
// last minute.
func stopObservability(ctx context.Context, shutdown telemetry.Shutdown, logger *slog.Logger) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), telemetryGrace)
	defer cancel()

	if err := shutdown(stopCtx); err != nil {
		logger.ErrorContext(stopCtx, "telemetry shutdown", slog.String("erreur", err.Error()))
	}
}

// serveMetrics exposes /metrics, fatal outside development.
//
// ⚠️ `telemetry.metrics_port` is ONE value, shared by both binaries. On the
// same host, with telemetry enabled — which is what
// `config/env/development.yaml` does —, the second one to start cannot bind.
// In development it therefore carries on, saying so; elsewhere it refuses.
//
// This is the same trade-off as in `cmd/server`, and it is justified in detail
// over there. The dispatcher adds a reason of its own: it is the only process
// whose inactivity nobody notices, hence the one whose metrics matter most —
// and the one for which refusing to start locally would cost the most in
// friction.
func serveMetrics(ctx context.Context, cfg config.Config, logger *slog.Logger, cancel context.CancelFunc) {
	if !cfg.Telemetry.Enabled {
		return
	}
	server := httpserver.NewMetricsServer(cfg.Telemetry.MetricsPort, logger)

	go func() {
		err := server.Run(ctx)
		if err == nil {
			return
		}
		fields := []any{
			slog.Int("port", cfg.Telemetry.MetricsPort),
			slog.String("erreur", err.Error()),
		}
		if cfg.App.Env.IsLocal() {
			logger.WarnContext(ctx, "metrics unavailable — the dispatcher continues (development)", fields...)
			return
		}
		logger.ErrorContext(ctx, "metrics server — stopping the dispatcher", fields...)
		cancel()
	}()
}
