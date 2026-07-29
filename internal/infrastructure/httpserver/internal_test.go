package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// These tests target UNEXPORTED fields: `Server.http` is private, and the
// timeouts it carries are precisely what has to be checked. This is the case
// provided for by rules/tests.md §2 for an `internal_test.go`.

func testConfig() config.Config {
	var cfg config.Config
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 0
	cfg.HTTP.ReadTimeout = config.Duration(3 * time.Second)
	cfg.HTTP.WriteTimeout = config.Duration(4 * time.Second)
	cfg.HTTP.IdleTimeout = config.Duration(5 * time.Second)
	cfg.HTTP.ShutdownTimeout = config.Duration(time.Second)
	return cfg
}

func quiet() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestEveryTimeoutIsSet: no timeout left at zero.
//
// # The defect this test catches: a one-line attack
//
// `ReadHeaderTimeout` at zero means "no limit". A connection that sends its
// headers one byte at a time then ties up a goroutine INDEFINITELY. A few
// thousand connections of that kind — a trivial script, no specialised tool —
// exhaust the server.
//
// This is the Slowloris attack, and its peculiarity is to look like nothing at
// all: no traffic spike, no error, no log. The service simply stops accepting
// connections.
//
// Go sets NONE of these timeouts by default. An `http.Server{}` written without
// thinking about it is vulnerable, and that is the most common case.
func TestEveryTimeoutIsSet(t *testing.T) {
	t.Parallel()

	server := New(testConfig(), nil, quiet())

	for name, got := range map[string]time.Duration{
		"ReadHeaderTimeout": server.http.ReadHeaderTimeout,
		"ReadTimeout":       server.http.ReadTimeout,
		"WriteTimeout":      server.http.WriteTimeout,
		"IdleTimeout":       server.http.IdleTimeout,
	} {
		if got == 0 {
			t.Errorf("%s is 0 — \"no limit\", therefore one goroutine per slow connection", name)
		}
	}
	// ReadHeaderTimeout must follow the configuration, not a forgotten constant.
	if server.http.ReadHeaderTimeout != 3*time.Second {
		t.Errorf("ReadHeaderTimeout = %s, want 3s (the configured value)",
			server.http.ReadHeaderTimeout)
	}
}

// TestMetricsServerListensOnLoopbackOnly: the metrics do not leave the machine.
//
// # Why this is a security requirement and not a tidiness one
//
// `/metrics` publishes the traffic volume, the route names, the latencies, the
// number of errors per type. It is a map of the internal structure and of the
// activity of the service, without authentication.
//
// Listening on `0.0.0.0` would expose it to anything that can reach the
// container. The binding is therefore explicitly `127.0.0.1`, and a collector
// reaches it through a side-car or a chosen redirection — an explicit decision,
// not an implicit default.
func TestMetricsServerListensOnLoopbackOnly(t *testing.T) {
	t.Parallel()

	server := NewMetricsServer(9100, quiet())

	if got := server.http.Addr; got != "127.0.0.1:9100" {
		t.Errorf("metrics address = %q, want 127.0.0.1:9100 — "+
			"any other binding publishes the internal structure of the service", got)
	}
	if server.http.ReadHeaderTimeout == 0 {
		t.Error("the metrics server has no ReadHeaderTimeout")
	}
}

// TestRunReturnsCleanlyOnCancellation: shutdown hands back, without an error.
//
// # The defect this test catches
//
// A `Run` that returned the `http.ErrServerClosed` error would make the NORMAL
// shutdown fail: the process would exit with a non-zero code at every
// deployment, the orchestrator would count it as a crash, and the deployment
// would be marked as failed although everything went well.
//
// The test binds to port 0 — the system picks a free one — so it conflicts with
// nothing, neither with another test nor with a service of the machine.
func TestRunReturnsCleanlyOnCancellation(t *testing.T) {
	t.Parallel()

	server := New(testConfig(), http.NotFoundHandler(), quiet())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()

	// Listening is asynchronous: we cancel without waiting, and Run must handle
	// both possible orders (cancelled before or after listening starts).
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v on shutdown — the deployment would be counted as failed", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not hand back: shutdown would block until SIGKILL")
	}
}
