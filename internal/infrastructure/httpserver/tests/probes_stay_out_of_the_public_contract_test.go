package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/infrastructure/httpserver"
)

// TestProbesStayOutOfThePublicContract: the probes are not in the OpenAPI.
//
// # Why it matters
//
// The OpenAPI contract is what the clients consume: SDK generators, portals,
// contract tests. The probes are no part of it — they are an operational
// detail, liable to change without notice.
//
// Were they to enter it, a generator would produce `healthz()` methods in every
// published SDK, and removing them would become a BREAKING change of contract.
// An operational decision would have turned into a public commitment, without
// anyone having wanted it.
//
// That is why they are mounted on the `*chi.Mux` directly, and not via
// `huma.Register`. This test locks that distinction, which does not show when
// reading `NewRouter`.
func TestProbesStayOutOfThePublicContract(t *testing.T) {
	t.Parallel()

	router := router(config.EnvProduction, map[string]httpserver.Probe{"database": healthyProbe()})

	// The OpenAPI description is built by huma from the routes REGISTERED with
	// it. The probes not being among them, it must be empty of any path.
	paths := router.API.OpenAPI().Paths
	for path := range paths {
		if strings.Contains(path, "healthz") || strings.Contains(path, "readyz") {
			t.Errorf("probe %q appears in the OpenAPI contract — a generated SDK would expose it", path)
		}
	}
}

// TestProbesRespondEvenThoughTheyAreUndocumented: out of contract ≠ absent.
//
// The counterpart of the previous test: a probe removed from the contract could
// also have been removed from the router. The two tests together say the only
// true thing — mounted, but not published.
func TestProbesRespondEvenThoughTheyAreUndocumented(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"/healthz", "/readyz"} {
		if code := get(t, config.EnvProduction, nil, path).Code; code == 404 {
			t.Errorf("%s returns 404 — the probe is not mounted", path)
		}
	}
}
