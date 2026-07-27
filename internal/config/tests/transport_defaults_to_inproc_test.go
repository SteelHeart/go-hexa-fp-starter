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
