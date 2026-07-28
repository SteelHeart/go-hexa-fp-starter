package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/auth"
)

// TestUnknownDriverRefusesStartup : deny par défaut jusque dans la fabrique.
//
// `postgres` et `oidc` sont annoncés par l'ADR 017 et ne sont PAS construits. Le
// refus doit être franc : un repli silencieux sur `memory` ferait tourner
// l'authentification d'une production sur un magasin volatile, qui EFFACE LES
// COMPTES au redémarrage — et rien ne le signalerait avant le premier
// redéploiement.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	c := newClock()
	for _, driver := range []string{"postgres", "oidc", "keycloak", "ldap", "memoire"} {
		if _, err := auth.New(
			config.Module{Enabled: true, Driver: driver}, deps(c),
		); err == nil {
			t.Errorf("le pilote %q n'est pas construit : il doit refuser le démarrage", driver)
		}
	}
}

// TestEmptyDriverFallsBackToTheDefault : ne rien préciser prend le pilote par
// défaut, et ce défaut n'exige aucune infrastructure.
//
// C'est ce qui rend vraie la promesse « `hexa new` puis `go run`, ça démarre » —
// y compris avec l'authentification ACTIVE.
func TestEmptyDriverFallsBackToTheDefault(t *testing.T) {
	t.Parallel()

	c := newClock()
	if _, err := auth.New(config.Module{Enabled: true}, deps(c)); err != nil {
		t.Fatalf("sans pilote précisé, le défaut doit s'appliquer : %v", err)
	}
}
