package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestTransportDefaultsToInproc : le mode le moins coûteux est le défaut. Un
// appel réseau ne s'active jamais par accident.
func TestTransportDefaultsToInproc(t *testing.T) {
	t.Parallel()

	if got := (config.Interop{}).TransportFor("user_registration"); got != "inproc" {
		t.Errorf("transport par défaut = %q, attendu \"inproc\"", got)
	}
}

func TestTransportPerModuleOverride(t *testing.T) {
	t.Parallel()

	interop := config.Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": "http", "search": "event"},
	}

	cases := map[string]string{
		"billing":           "http",
		"search":            "event",
		"user_registration": "inproc",
	}
	for module, want := range cases {
		if got := interop.TransportFor(module); got != want {
			t.Errorf("transport de %s = %q, attendu %q", module, got, want)
		}
	}
}

// TestTransportIgnoresEmptyOverride : une entrée vide dans la configuration ne
// doit pas désactiver silencieusement le transport — elle retombe sur le défaut.
func TestTransportIgnoresEmptyOverride(t *testing.T) {
	t.Parallel()

	interop := config.Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": ""},
	}
	if got := interop.TransportFor("billing"); got != "inproc" {
		t.Errorf("surcharge vide = %q, attendu le défaut \"inproc\"", got)
	}
}

// TestTransportSwitchIsPureConfiguration documente l'intention de l'ADR 010 :
// passer un module de inproc à http, c'est l'extraire en service sans toucher
// une ligne de code. Ce test échouerait si la résolution devenait implicite.
func TestTransportSwitchIsPureConfiguration(t *testing.T) {
	t.Parallel()

	before := config.Interop{DefaultTransport: "inproc"}
	after := config.Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"user_registration": "http"},
		BaseURLs:         map[string]string{"user_registration": "http://user-registration:8080"},
	}

	if before.TransportFor("user_registration") != "inproc" {
		t.Fatal("état initial inattendu")
	}
	if after.TransportFor("user_registration") != "http" {
		t.Error("le changement de mode doit venir de la seule configuration")
	}
}
