// Package telemetry wires the traces, the metrics and the logs, and links them
// through the trace_id.
//
// A log without a trace_id is a log that cannot be cross-checked during an
// incident: that is why the slog handler injects it itself rather than relying
// on the callers.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// Shutdown releases the exporters. To be called on a NON-cancelled context,
// otherwise the buffered spans are lost at the very moment they are the most
// useful.
type Shutdown = func(context.Context) error

// NewLogger builds the structured log.
//
// JSON outside development: that is what the collectors know how to read. Text
// locally, because a human reads text better.
func NewLogger(cfg config.Config) *slog.Logger {
	return newLoggerTo(cfg, os.Stdout)
}

// newLoggerTo carries the whole logic, the destination being a parameter.
//
// The destination is extracted for ONE reason: a hard-coded `os.Stdout` made
// the only interesting rule of this function unverifiable — outside
// development, the format is JSON even if the configuration asks for text. A
// rule that no test can observe is not a rule.
func newLoggerTo(cfg config.Config, out io.Writer) *slog.Logger {
	level := parseLevel(cfg.Telemetry.LogLevel)
	opts := &slog.HandlerOptions{Level: level, AddSource: level == slog.LevelDebug}

	var base slog.Handler
	if cfg.Telemetry.LogFormat == "json" || !cfg.App.Env.IsDevelopment() {
		base = slog.NewJSONHandler(out, opts)
	} else {
		base = slog.NewTextHandler(out, opts)
	}

	handler := &traceHandler{inner: base}
	return slog.New(handler).With(
		slog.String("service", cfg.App.Name),
		slog.String("env", string(cfg.App.Env)),
		slog.String("version", cfg.App.Version),
	)
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(raw) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// traceHandler injects trace_id and span_id into every record.
type traceHandler struct{ inner slog.Handler }

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *traceHandler) Handle(ctx context.Context, record slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, record) //nolint:wrapcheck // direct delegation
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{inner: h.inner.WithGroup(name)}
}

// Setup installs the trace and metric providers.
//
// Disabled, the function returns an inert shutdown: the calling code does not
// have to know whether telemetry is active, and OpenTelemetry's no-op providers
// make every call free.
func Setup(ctx context.Context, cfg config.Config) (Shutdown, error) {
	if !cfg.Telemetry.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", cfg.Telemetry.ServiceName),
		attribute.String("service.version", cfg.App.Version),
		attribute.String("deployment.environment", string(cfg.App.Env)),
	))
	if err != nil {
		return nil, fmt.Errorf("OpenTelemetry resource: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Telemetry.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	metricExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("prometheus exporter: %w", err)
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricExporter),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	return func(shutdownCtx context.Context) error {
		// BOTH shutdowns are attempted, then their errors are joined.
		//
		// Two defects lived here, both invisible without a test:
		//
		//  1. The `fmt.Errorf(... %w ...)` was UNCONDITIONAL. With a `%w` on a
		//     nil, it does not return nil: it returns an error carrying
		//     "%!w(<nil>)". A PERFECTLY SUCCESSFUL shutdown therefore returned
		//     an error, and the caller would have counted every normal
		//     deployment as a failure.
		//  2. The helper was called `errJoin` and joined nothing: it returned
		//     the FIRST non-nil error and threw away the second. If the metric
		//     exporter failed to flush after the trace one, its error
		//     disappeared — and an exporter that does not flush loses the
		//     measurements of the very incident one is trying to understand.
		//
		// `errors.Join` does exactly what the name announced, and returns nil
		// when all is well. The home-made helper had no reason to exist.
		if err := errors.Join(
			tracerProvider.Shutdown(shutdownCtx),
			meterProvider.Shutdown(shutdownCtx),
		); err != nil {
			return fmt.Errorf("telemetry shutdown: %w", err)
		}
		return nil
	}, nil
}
