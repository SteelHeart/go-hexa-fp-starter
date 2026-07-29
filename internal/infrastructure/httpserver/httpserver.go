// Package httpserver mounts the router, the documented API and the graceful
// shutdown.
//
// It is the ONLY package that knows chi and huma outside the HTTP primary
// adapters. The cost of leaving the framework therefore fits in this file
// (documentation/adr/008).
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/pkg/middleware"
)

// Probe reports the health of a dependency. It returns an explicit error so
// that /readyz can say WHAT is wrong.
type Probe = func(context.Context) error

// Router carries the router and the documented API.
type Router struct {
	Mux *chi.Mux
	API huma.API
}

// NewRouter builds the router, the middleware stack and the probes.
//
// The order of the middlewares is significant and reads from the outside
// inwards: the correlation identifier must exist before anything at all logs,
// and the panic recovery must wrap all the rest.
func NewRouter(cfg config.Config, logger *slog.Logger, readiness map[string]Probe) *Router {
	mux := chi.NewMux()

	limiter := middleware.NewRateLimiter(cfg.Limits.RPS, cfg.Limits.Burst, 10*time.Minute)
	mux.Use(
		middleware.RequestID(),
		middleware.Recover(logger),
		securityHeadersFor(cfg.App.Env),
		middleware.CORS(cfg.HTTP.AllowedOrigins),
		middleware.MaxBody(cfg.HTTP.MaxBodyBytes),
		middleware.AccessLog(logger),
		limiter.Middleware(),
	)

	mountProbes(mux, readiness)

	humaCfg := huma.DefaultConfig(cfg.App.Name, cfg.App.Version)
	humaCfg.DocsPath = "/docs"
	humaCfg.OpenAPIPath = "/openapi"
	return &Router{Mux: mux, API: humachi.New(mux, humaCfg)}
}

// securityHeadersFor picks the header set according to the environment.
//
// Deny by default: ONLY development gets the version without HSTS, and it must
// name itself to get it. An unknown environment — therefore misconfigured —
// receives the full hardening.
func securityHeadersFor(env config.Environment) middleware.Middleware {
	if env.IsDevelopment() {
		return middleware.SecurityHeadersWithoutHSTS()
	}
	return middleware.SecurityHeaders()
}

// mountProbes mounts the probes outside the documented API: they are not part
// of the public contract and must not appear in the OpenAPI.
func mountProbes(mux *chi.Mux, readiness map[string]Probe) {
	// /healthz checks NO dependency: otherwise a database incident would
	// restart every container, turning a partial outage into a total
	// unavailability.
	mux.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		for name, probe := range readiness {
			if err := probe(ctx); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = fmt.Fprintf(w, `{"status":"unavailable","dependency":%q}`, name)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
}

// Server wraps the HTTP server and its graceful shutdown.
type Server struct {
	http   *http.Server
	logger *slog.Logger
	grace  time.Duration
}

// New builds the application server.
//
// otelhttp wraps the handler: each request opens a root span, which then links
// together all the logs and spans of the request.
func New(cfg config.Config, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		http: &http.Server{
			Addr:    cfg.HTTP.Addr(),
			Handler: otelhttp.NewHandler(handler, "http.server"),
			// Non-zero ReadHeaderTimeout: without it, a connection that never
			// sends its headers ties up a goroutine indefinitely.
			ReadHeaderTimeout: cfg.HTTP.ReadTimeout.Duration(),
			ReadTimeout:       cfg.HTTP.ReadTimeout.Duration(),
			WriteTimeout:      cfg.HTTP.WriteTimeout.Duration(),
			IdleTimeout:       cfg.HTTP.IdleTimeout.Duration(),
		},
		logger: logger,
		grace:  cfg.HTTP.ShutdownTimeout.Duration(),
	}
}

// Run listens until the context is cancelled, then drains the in-flight
// connections.
func (s *Server) Run(ctx context.Context) error {
	errs := make(chan error, 1)
	go func() {
		s.logger.InfoContext(ctx, "HTTP server listening", slog.String("addr", s.http.Addr))
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
			return
		}
		errs <- nil
	}()

	select {
	case err := <-errs:
		if err != nil {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	case <-ctx.Done():
		return s.shutdown(ctx)
	}
}

func (s *Server) shutdown(ctx context.Context) error {
	// WithoutCancel rather than Background: the parent context is already
	// cancelled and using it as is would cut the in-flight requests instead of
	// letting them finish — but starting again from Background would ALSO throw
	// away the values carried by the context, including the trace. Shutdown
	// would then be the only moment of the lifecycle invisible in the
	// observability, that is to say precisely the one we are trying to
	// understand after an incident.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.grace)
	defer cancel()
	s.logger.InfoContext(shutdownCtx, "shutting down the HTTP server", slog.Duration("grace", s.grace))
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("HTTP server shutdown: %w", err)
	}
	return nil
}

// NewMetricsServer exposes /metrics on a separate port.
//
// Separate port and not publicly exposed: the metrics reveal the traffic volume
// and the internal structure of the service.
func NewMetricsServer(port int, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return &Server{
		http: &http.Server{
			Addr:              fmt.Sprintf("127.0.0.1:%d", port),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
		logger: logger,
		grace:  5 * time.Second,
	}
}
