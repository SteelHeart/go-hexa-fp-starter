package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// Tests of the NON EXPORTED identifiers of the composition root.
//
// `{package}/internal_test.go` and not `{package}/tests/`: `rules/tests.md` §
// "Black box, white box" reserves the latter for the public API of a package,
// and Go forbids importing a `main` package. A black-box test of this wiring
// is therefore impossible without taking the wiring out of the composition
// root — which is not wanted, since that is precisely its role (ADR 004).

// quietLogger returns a logger that writes nowhere.
//
// These tests assert on BEHAVIOUR — a port that answers, a context that gets
// cancelled — never on log text: a test that asserts on wording breaks at the
// first rephrasing and proves nothing more.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// freePort reserves a port then releases it, so parallel tests never collide.
func freePort(t *testing.T) int {
	t.Helper()
	listener := listen(t)
	port := portOf(t, listener)
	if err := listener.Close(); err != nil {
		t.Fatalf("releasing the port: %v", err)
	}
	return port
}

// listen opens a listener on an arbitrary free port.
func listen(t *testing.T) net.Listener {
	t.Helper()
	var lc net.ListenConfig
	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserving a port: %v", err)
	}
	return listener
}

func portOf(t *testing.T, listener net.Listener) int {
	t.Helper()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected a TCP address, got %T", listener.Addr())
	}
	return addr.Port
}

// TestDisabledTelemetryOpensNoPort: telemetry off, no extra port.
//
// This is the ADR 012 promise applied to the observability wiring: plugging it
// in must add NO prerequisite. A port opened without anyone asking for it is an
// attack surface that would appear without a decision.
func TestDisabledTelemetryOpensNoPort(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	cfg := config.Config{Telemetry: config.Telemetry{Enabled: false, MetricsPort: port}}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	serveMetrics(ctx, cfg, quietLogger(), cancel)

	// Give the server the time it would need to bind: without this wait, the
	// test would pass just as well because nothing listens as because we looked
	// too early.
	time.Sleep(150 * time.Millisecond)

	if answers(port) {
		t.Errorf("a metrics port is open on %d while telemetry is disabled", port)
	}
}

// TestEnabledTelemetryServesMetrics: telemetry on, /metrics answers.
//
// The positive half of the guard. Without it, this change could be green having
// wired nothing at all — the exact defect it repairs.
func TestEnabledTelemetryServesMetrics(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	cfg := config.Config{Telemetry: config.Telemetry{Enabled: true, MetricsPort: port}}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	serveMetrics(ctx, cfg, quietLogger(), cancel)

	if !waitFor(port) {
		t.Fatalf("/metrics does not answer on %d", port)
	}
}

// TestABusyPortStopsTheServiceOutsideDevelopment: in production, a metrics port
// already taken stops the service.
//
// This is the case that makes the guard fail (ADR 013). Without it, "refuses
// correctly" and "never tried anything" produce exactly the same output — and
// this change ADDS a guard, so it must show that it bites.
func TestABusyPortStopsTheServiceOutsideDevelopment(t *testing.T) {
	t.Parallel()

	occupant := listen(t)
	defer func() { _ = occupant.Close() }()

	cfg := config.Config{
		App:       config.App{Env: config.EnvProduction},
		Telemetry: config.Telemetry{Enabled: true, MetricsPort: portOf(t, occupant)},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	serveMetrics(ctx, cfg, quietLogger(), cancel)

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Error("a busy metrics port had to stop the service outside development")
	}
}

// TestABusyPortDoesNotStopDevelopment: locally, the same failure lets the
// service run.
//
// The waiver is NAMED and bounded, like `SecurityHeadersWithoutHSTS` elsewhere
// in the starter. It exists because `config/env/development.yaml` enables
// telemetry: the port is therefore open by default locally, and a machine where
// 9090 is already taken could no longer run `go run` — over a diagnostics port.
// Measured while wiring, not anticipated.
func TestABusyPortDoesNotStopDevelopment(t *testing.T) {
	t.Parallel()

	occupant := listen(t)
	defer func() { _ = occupant.Close() }()

	cfg := config.Config{
		App:       config.App{Env: config.EnvDevelopment},
		Telemetry: config.Telemetry{Enabled: true, MetricsPort: portOf(t, occupant)},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	serveMetrics(ctx, cfg, quietLogger(), cancel)

	select {
	case <-ctx.Done():
		t.Error("in development a busy port must not stop the service")
	case <-time.After(time.Second):
	}
}

// answers reports whether /metrics answers on this port.
func answers(port int) bool {
	client := &http.Client{Timeout: 300 * time.Millisecond}
	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodGet, metricsURL(port), http.NoBody)
	if err != nil {
		return false
	}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = res.Body.Close()
	return res.StatusCode == http.StatusOK
}

// waitFor gives the server time to bind.
func waitFor(port int) bool {
	for range 40 {
		if answers(port) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func metricsURL(port int) string {
	return "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/metrics"
}
