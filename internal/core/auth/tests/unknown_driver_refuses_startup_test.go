package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// TestUnknownDriverRefusesStartup: deny by default all the way into the
// factory.
//
// `postgres` and `oidc` are announced by ADR 017 and are NOT built. The refusal
// must be blunt: a silent fallback on `memory` would run a production's
// authentication on a volatile store, which ERASES THE ACCOUNTS on restart —
// and nothing would report it before the first redeployment.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	c := newClock()
	for _, driver := range []string{"postgres", "oidc", "keycloak", "ldap", "memoy"} {
		if _, err := auth.New(
			config.Module{Enabled: true, Driver: driver}, deps(c),
		); err == nil {
			t.Errorf("driver %q is not built: it must refuse startup", driver)
		}
	}
}

// TestEmptyDriverFallsBackToTheDefault: specifying nothing takes the default
// driver, and that default demands no infrastructure.
//
// That is what makes the promise "`hexa new` then `go run`, and it starts"
// true — including with authentication ENABLED.
func TestEmptyDriverFallsBackToTheDefault(t *testing.T) {
	t.Parallel()

	c := newClock()
	if _, err := auth.New(config.Module{Enabled: true}, deps(c)); err != nil {
		t.Fatalf("with no driver specified, the default must apply: %v", err)
	}
}
