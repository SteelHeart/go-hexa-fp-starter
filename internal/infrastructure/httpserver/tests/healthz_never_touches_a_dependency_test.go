package tests

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

// TestHealthzNeverTouchesADependency: /healthz is INDEPENDENT of the dependencies.
//
// # The defect this test catches, and it turns a partial outage into a total one
//
// It is the most tempting mistake of the whole HTTP mounting: "we may as well
// check the database in /healthz too, it is more complete". It is catastrophic.
//
// /healthz is the LIVENESS probe: the orchestrator kills and restarts the
// container that answers it badly. If it queries the database, then a database
// outage makes the probe fail on ALL the replicas at once. Kubernetes kills
// them all, restarts them, they fail again, it kills them again.
//
// Result: a momentarily unavailable database — from which one recovers —
// becomes an entirely dead service, in a restart loop, incapable even of
// serving what did not depend on the database. The remedy worsens the illness.
//
// The distinction is sharp and must stay tooled:
//
//	/healthz  am I ALIVE?        no dependency         → otherwise I get killed
//	/readyz   can I SERVE?       every dependency      → otherwise I am pulled out of the traffic
func TestHealthzNeverTouchesADependency(t *testing.T) {
	t.Parallel()

	var probeCalls atomic.Int64
	watched := map[string]httpserver.Probe{
		// A probe that counts its calls AND that fails: if /healthz consulted
		// it, the test would see it twice over.
		"database": func(context.Context) error {
			probeCalls.Add(1)
			return errNotAvailable
		},
	}

	rec := get(t, config.EnvProduction, watched, "/healthz")

	if rec.Code != http.StatusOK {
		t.Errorf("/healthz = %d while a dependency is down, want 200 — "+
			"the orchestrator would kill every replica in a loop", rec.Code)
	}
	if calls := probeCalls.Load(); calls != 0 {
		t.Errorf("/healthz consulted %d probe(s) — it must consult NONE", calls)
	}
}

// TestHealthzAnswersJSON: the body is JSON, with its content type.
//
// A probe that returns raw text announced as `text/plain` breaks the collectors
// that deserialise the response — and nobody notices before the monitoring is
// put in place, that is to say too late.
func TestHealthzAnswersJSON(t *testing.T) {
	t.Parallel()

	rec := get(t, config.EnvProduction, nil, "/healthz")

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if body := rec.Body.String(); body != `{"status":"ok"}` {
		t.Errorf("body = %q", body)
	}
}
