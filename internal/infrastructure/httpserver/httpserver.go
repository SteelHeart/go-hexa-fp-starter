// Package httpserver monte le routeur, l'API documentÃ©e et l'arrÃªt propre.
//
// C'est le SEUL paquet qui connaÃ®t chi et huma en dehors des adaptateurs
// primaires HTTP. Le coÃ»t de sortie du framework tient donc dans ce fichier
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

// Probe rapporte la santÃ© d'une dÃ©pendance. Elle retourne une erreur explicite
// pour que /readyz puisse dire CE QUI ne va pas.
type Probe = func(context.Context) error

// Router porte le routeur et l'API documentÃ©e.
type Router struct {
	Mux *chi.Mux
	API huma.API
}

// NewRouter construit le routeur, la pile de middlewares et les sondes.
//
// L'ordre des middlewares est significatif et se lit de l'extÃ©rieur vers
// l'intÃ©rieur : l'identifiant de corrÃ©lation doit exister avant que quoi que ce
// soit ne journalise, et la rÃ©cupÃ©ration de panique doit envelopper tout le reste.
func NewRouter(cfg config.Config, logger *slog.Logger, readiness map[string]Probe) *Router {
	mux := chi.NewMux()

	limiter := middleware.NewRateLimiter(cfg.Limits.RPS, cfg.Limits.Burst, 10*time.Minute)
	mux.Use(
		middleware.RequestID(),
		middleware.Recover(logger),
		middleware.SecurityHeaders(!cfg.App.Env.IsDevelopment()),
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

// mountProbes monte les sondes hors de l'API documentÃ©e : elles ne font pas
// partie du contrat public et ne doivent pas apparaÃ®tre dans l'OpenAPI.
func mountProbes(mux *chi.Mux, readiness map[string]Probe) {
	// /healthz ne vÃ©rifie AUCUNE dÃ©pendance : sinon un incident base ferait
	// redÃ©marrer tous les conteneurs, transformant une panne partielle en
	// indisponibilitÃ© totale.
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

// Server encapsule le serveur HTTP et son arrÃªt propre.
type Server struct {
	http   *http.Server
	logger *slog.Logger
	grace  time.Duration
}

// New construit le serveur applicatif.
//
// otelhttp enveloppe le gestionnaire : chaque requÃªte ouvre un span racine, ce
// qui relie ensuite tous les logs et spans de la requÃªte.
func New(cfg config.Config, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		http: &http.Server{
			Addr:    cfg.HTTP.Addr(),
			Handler: otelhttp.NewHandler(handler, "http.server"),
			// ReadHeaderTimeout non nul : sans lui, une connexion qui n'envoie
			// jamais ses en-tÃªtes immobilise une goroutine indÃ©finiment.
			ReadHeaderTimeout: cfg.HTTP.ReadTimeout.Duration(),
			ReadTimeout:       cfg.HTTP.ReadTimeout.Duration(),
			WriteTimeout:      cfg.HTTP.WriteTimeout.Duration(),
			IdleTimeout:       cfg.HTTP.IdleTimeout.Duration(),
		},
		logger: logger,
		grace:  cfg.HTTP.ShutdownTimeout.Duration(),
	}
}

// Run Ã©coute jusqu'Ã  l'annulation du contexte, puis vide les connexions en cours.
func (s *Server) Run(ctx context.Context) error {
	errs := make(chan error, 1)
	go func() {
		s.logger.InfoContext(ctx, "serveur HTTP Ã  l'Ã©coute", slog.String("addr", s.http.Addr))
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- err
			return
		}
		errs <- nil
	}()

	select {
	case err := <-errs:
		if err != nil {
			return fmt.Errorf("serveur HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		return s.shutdown()
	}
}

func (s *Server) shutdown() error {
	// Contexte dÃ©tachÃ© : le contexte parent est dÃ©jÃ  annulÃ©, l'utiliser ici
	// couperait les requÃªtes en cours au lieu de les laisser finir.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.grace)
	defer cancel()
	s.logger.Info("arrÃªt du serveur HTTP", slog.Duration("grace", s.grace))
	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("arrÃªt du serveur HTTP: %w", err)
	}
	return nil
}

// NewMetricsServer expose /metrics sur un port sÃ©parÃ©.
//
// Port sÃ©parÃ© et non exposÃ© publiquement : les mÃ©triques rÃ©vÃ¨lent la volumÃ©trie
// et la structure interne du service.
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
