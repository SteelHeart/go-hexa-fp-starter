package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

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
