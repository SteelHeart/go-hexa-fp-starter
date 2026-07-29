// Package tests exercises the HTTP mounting through its PUBLIC API.
//
// This package mounts the stack that EVERY request goes through, and the two
// probes the orchestrator queries to decide whether to kill a container or to
// send it traffic. A mistake here does not translate into a functional bug: it
// translates into unavailability, or into a leak in a response body.
//
// None of this requires a service: the router is built in memory and the probes
// are closures.
package tests

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

// quietLogger returns a silent logger: these tests observe RESPONSES, not
// traces.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(newDiscard(), nil))
}

// newDiscard avoids importing io.Discard in every file.
func newDiscard() *discard { return &discard{} }

type discard struct{}

func (*discard) Write(p []byte) (int, error) { return len(p), nil }

// serverConfig builds a configuration sufficient to mount the router.
//
// The bounds are wide: these tests do not measure rate limiting, it has its own
// tests in `internal/pkg/middleware/tests`. A rate too low here would make
// tests fail for a reason that is not theirs.
func serverConfig(env config.Environment) config.Config {
	var cfg config.Config
	cfg.App.Env = env
	cfg.App.Name = "hexa-tests"
	cfg.App.Version = "0.0.0-tests"
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 0
	cfg.HTTP.MaxBodyBytes = 1 << 20
	cfg.HTTP.AllowedOrigins = []string{"https://app.example.com"}
	cfg.HTTP.ReadTimeout = config.Duration(2 * time.Second)
	cfg.HTTP.WriteTimeout = config.Duration(2 * time.Second)
	cfg.HTTP.IdleTimeout = config.Duration(2 * time.Second)
	cfg.HTTP.ShutdownTimeout = config.Duration(time.Second)
	cfg.Limits.RPS = 1000
	cfg.Limits.Burst = 1000
	return cfg
}

// router mounts the router with the supplied probes.
func router(env config.Environment, readiness map[string]httpserver.Probe) *httpserver.Router {
	return httpserver.NewRouter(serverConfig(env), quietLogger(), readiness)
}

// routerWith mounts the router from a configuration the caller has adjusted.
//
// `router` above starts from `serverConfig`, which is deliberately wide so that
// no test fails for a bound that is not its subject. A test whose subject IS a
// bound needs to set it, hence this second entry point.
func routerWith(cfg config.Config) *httpserver.Router {
	return httpserver.NewRouter(cfg, quietLogger(), nil)
}

// get queries a path of the router and returns the recorded response.
func get(t *testing.T, env config.Environment, readiness map[string]httpserver.Probe, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "http://127.0.0.1"+path, nil)
	rec := httptest.NewRecorder()
	router(env, readiness).Mux.ServeHTTP(rec, req)
	return rec
}

// failingProbe returns a probe that fails with the given error.
func failingProbe(message string) httpserver.Probe {
	return func(context.Context) error { return errors.New(message) }
}

// healthyProbe returns a probe that succeeds.
func healthyProbe() httpserver.Probe {
	return func(context.Context) error { return nil }
}
