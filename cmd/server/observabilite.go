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

// ─────────────────────────────────────────────────────────────────────────────
// This file is the half that was missing — #13
// ─────────────────────────────────────────────────────────────────────────────
//
// `telemetry.Setup` and `httpserver.NewMetricsServer` existed, were tested,
// and were called by NO binary. Consequence measured before the fix:
//
//	occurrences of trace_id in the logs : 0
//	GET /metrics                        : no answer
//
// `otelhttp` did already wrap the router, and the slog handler did already
// inject `trace_id` — but with no trace provider installed, the span opened by
// `otelhttp` is not recorded, its context is invalid, and the injection never
// fires. Three pieces out of four, hence nothing.
//
// This is the failure mode that `documentation/produit/personas.md` names as
// the worst one for the operator: *an observability configuration without
// wiring is worse than none, it makes people believe the question is handled.*

// telemetryGrace bounds the shutdown of the exporters.
//
// A finite delay rather than none: an unreachable collector would otherwise
// make the shutdown wait indefinitely, that is, turn an observability incident
// into an availability incident.
const telemetryGrace = 5 * time.Second

// startObservability installs the providers and returns the shutdown function.
//
// Disabled — the shipped default —, `Setup` returns an inert shutdown: this
// wiring therefore adds NO infrastructure prerequisite. `go run ./cmd/server`
// still starts with no collector, no database and no Docker (ADR 012).
func startObservability(ctx context.Context, cfg config.Config) (telemetry.Shutdown, error) {
	shutdown, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("telemetry: %w", err)
	}
	return shutdown, nil
}

// stopObservability flushes the exporters before handing back control.
//
// The context is deliberately DETACHED from the cancellation: at shutdown time
// `ctx` is already cancelled, and passing it as is would drop the buffered
// spans — those of the last minute, that is, precisely the ones people will
// look for after an abnormal shutdown.
func stopObservability(ctx context.Context, shutdown telemetry.Shutdown, logger *slog.Logger) {
	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), telemetryGrace)
	defer cancel()

	if err := shutdown(stopCtx); err != nil {
		logger.ErrorContext(stopCtx, "telemetry shutdown", slog.String("erreur", err.Error()))
	}
}

// serveMetrics exposes /metrics, and decides what to do when the port is taken.
//
// # The failure is fatal OUTSIDE development, and that is a measured trade-off
//
// In production, a service running without metrics is exactly what nobody
// notices before the incident where they are needed. The refusal must
// therefore be total.
//
// In development, no — and that is a correction, not a tolerance. The previous
// version cut everywhere. Yet `config/env/development.yaml` enables telemetry:
// port 9090 is therefore open by DEFAULT locally, and 9090 is taken on many
// workstations. "`go run` starts on a bare machine" would have stopped being
// true because of a diagnostics port. Measured by wiring it, not anticipated.
//
// The shape follows the one already retained elsewhere in the starter —
// `SecurityHeaders()` against `SecurityHeadersWithoutHSTS()`: the strict
// behaviour is the default, and the waiver is NAMED and bounded. Here it is
// bounded to `development`/`test`, and it is noisy.
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
			logger.WarnContext(ctx, "metrics unavailable — the service continues (development)", fields...)
			return
		}
		logger.ErrorContext(ctx, "metrics server — stopping the service", fields...)
		cancel()
	}()
}
