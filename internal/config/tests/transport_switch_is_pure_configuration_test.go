package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

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
