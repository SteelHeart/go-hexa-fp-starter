package tests

import (
	"net/http"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestADisabledModuleAnswers503Not200 guards the worst default scenario.
//
// # What is really being checked
//
// That a turned-off module REFUSES, rather than letting things through. The
// surface mounts anyway — that is what makes it possible to answer clearly
// instead of failing the whole server startup — and the temptation is then to
// write the refusals "that matter" while forgetting the others.
//
// 503 and not 501: the capability EXISTS, it is not enabled on this deployment.
// That is an operational decision, not a missing feature, and 503 is what
// allows a client to retry later.
func TestADisabledModuleAnswers503Not200(t *testing.T) {
	t.Parallel()

	server := serve(t, newModule(t, config.Module{Enabled: false}))
	token := "0123456789012345678901234567890123456789012"

	cases := map[string]response{
		"opening": openSession(t, server, subject, secret),
		"resolving": withBearer(
			t, server, http.MethodGet, identityPath, "Bearer "+token),
		"closing": withBearer(
			t, server, http.MethodDelete, currentPath, "Bearer "+token),
	}

	for name, resp := range cases {
		if resp.status != http.StatusServiceUnavailable {
			t.Errorf("%s on a disabled module: want 503, got %d — %s", name, resp.status, resp.raw)
		}
	}
}
