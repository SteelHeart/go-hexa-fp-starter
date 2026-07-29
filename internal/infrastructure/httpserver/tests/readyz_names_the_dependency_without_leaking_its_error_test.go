package tests

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

// errNotAvailable is the probe failure shared by these tests.
var errNotAvailable = errors.New("unavailable")

// TestReadyzNamesTheDependencyWithoutLeakingItsError: the NAME, never the CAUSE.
//
// # The defect this test catches — and it is a secret leak
//
// The reflex, when writing this probe, is to include the error: "otherwise we
// do not know why". Yet the error of a database probe commonly contains the
// CONNECTION STRING, password included — that is what pgx returns when it fails
// to connect.
//
// /readyz is queried by the orchestrator, but nothing protects it: it is an
// unauthenticated HTTP entry point, mounted before any authorisation, and often
// reachable from the internal network. Writing the raw error there publishes
// the database password to anyone who can make a GET.
//
// The correct output names the dependency — enough to act — and nothing more.
// The cause, for its part, goes into the logs, whose access is controlled.
func TestReadyzNamesTheDependencyWithoutLeakingItsError(t *testing.T) {
	t.Parallel()

	// A realistic probe error: this is the shape a pgx failure takes.
	const dsnish = "failed to connect to " +
		"postgres://hexa_app:sup3rs3cr3t@db.internal:5432/hexa: connection refused"

	broken := map[string]httpserver.Probe{"database": failingProbe(dsnish)}

	rec := get(t, config.EnvProduction, broken, "/readyz")
	body := rec.Body.String()

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz = %d with a dependency down, want 503", rec.Code)
	}
	if !strings.Contains(body, "database") {
		t.Errorf("the failing dependency is not named: %q — the probe is unusable", body)
	}
	for _, secret := range []string{"sup3rs3cr3t", "hexa_app", "db.internal", "connection refused"} {
		if strings.Contains(body, secret) {
			t.Errorf("LEAK: %q appears in the body of /readyz: %s", secret, body)
		}
	}
}

// TestReadyzIsReadyWhenEveryProbePasses: all green → 200.
//
// Without this test, the probe could return 503 permanently and the 503 test
// would stay green. A service that is never ready never receives traffic: the
// deployment fails instead of completing, and nobody understands why.
func TestReadyzIsReadyWhenEveryProbePasses(t *testing.T) {
	t.Parallel()

	healthy := map[string]httpserver.Probe{
		"database": healthyProbe(),
		"cache":    healthyProbe(),
		"outbox":   healthyProbe(),
	}

	rec := get(t, config.EnvProduction, healthy, "/readyz")

	if rec.Code != http.StatusOK {
		t.Errorf("/readyz = %d with every probe green, want 200 — "+
			"the service would never receive any traffic", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "ready") {
		t.Errorf("body = %q", body)
	}
}

// TestReadyzWithoutProbesIsReady: no probe declared → ready.
//
// This is the configuration of a freshly generated starter, on its default
// drivers: neither database nor cache, therefore nothing to probe. `hexa new`
// then `go run` must pass /readyz, otherwise a user's first deployment fails
// without them having written a single line.
func TestReadyzWithoutProbesIsReady(t *testing.T) {
	t.Parallel()

	for name, readiness := range map[string]map[string]httpserver.Probe{
		"nil map":   nil,
		"empty map": {},
	} {
		if rec := get(t, config.EnvProduction, readiness, "/readyz"); rec.Code != http.StatusOK {
			t.Errorf("%s: /readyz = %d, want 200", name, rec.Code)
		}
	}
}
