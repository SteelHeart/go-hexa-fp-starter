// Package telemetry câble les traces, les métriques et les logs, et les relie
// par le trace_id.
//
// Un log sans trace_id est un log qu'on ne pourra pas recouper en incident :
// c'est pour cela que le gestionnaire slog l'injecte lui-même plutôt que de
// compter sur les appelants.
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

// Shutdown libère les exportateurs. À appeler sur un contexte NON annulé,
// sinon les spans en tampon sont perdus au moment où ils sont les plus utiles.
type Shutdown = func(context.Context) error

// NewLogger construit le journal structuré.
//
// JSON hors développement : c'est ce que les collecteurs savent lire. Texte en
// local, parce qu'un humain lit mieux du texte.
func NewLogger(cfg config.Config) *slog.Logger {
	return newLoggerTo(cfg, os.Stdout)
}

// newLoggerTo porte toute la logique, la destination étant un paramètre.
//
// La destination est extraite pour UNE raison : `os.Stdout` en dur rendait
// invérifiable la seule règle intéressante de cette fonction — hors
// développement, le format est JSON même si la configuration demande du texte.
// Une règle qu'aucun test ne peut observer n'est pas une règle.
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

// traceHandler injecte trace_id et span_id dans chaque enregistrement.
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
	return h.inner.Handle(ctx, record) //nolint:wrapcheck // délégation directe
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{inner: h.inner.WithGroup(name)}
}

// Setup installe les fournisseurs de traces et de métriques.
//
// Désactivée, la fonction retourne un arrêt inerte : le code appelant n'a pas Ã
// savoir si la télémétrie est active, et les no-op providers d'OpenTelemetry
// rendent tout appel gratuit.
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
		return nil, fmt.Errorf("ressource OpenTelemetry: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Telemetry.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("exportateur de traces: %w", err)
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
		return nil, fmt.Errorf("exportateur Prometheus: %w", err)
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricExporter),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	return func(shutdownCtx context.Context) error {
		// Les DEUX arrêts sont tentés, puis leurs erreurs sont jointes.
		//
		// Deux défauts vivaient ici, tous deux invisibles sans test :
		//
		//  1. Le `fmt.Errorf(... %w ...)` était INCONDITIONNEL. Avec un `%w` sur
		//     un nil, il ne rend pas nil : il rend une erreur portant
		//     « %!w(<nil>) ». Un arrêt PARFAITEMENT RÉUSSI remontait donc une
		//     erreur, et l'appelant aurait compté chaque déploiement normal comme
		//     un échec.
		//  2. L'aide s'appelait `errJoin` et ne joignait rien : elle rendait la
		//     PREMIÈRE erreur non nulle et jetait la seconde. Si l'exportateur de
		//     métriques échouait à se vider après celui des traces, son erreur
		//     disparaissait — et un exportateur qui ne se vide pas perd les
		//     mesures de l'incident qu'on cherche justement à comprendre.
		//
		// `errors.Join` fait exactement ce qu'annonçait le nom, et rend nil quand
		// tout va bien. L'aide maison n'avait aucune raison d'exister.
		if err := errors.Join(
			tracerProvider.Shutdown(shutdownCtx),
			meterProvider.Shutdown(shutdownCtx),
		); err != nil {
			return fmt.Errorf("arrêt de la télémétrie: %w", err)
		}
		return nil
	}, nil
}
