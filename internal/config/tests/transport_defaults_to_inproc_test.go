package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestTransportDefaultsToInproc: the least costly mode is the default. A
// network call never enables itself by accident.
func TestTransportDefaultsToInproc(t *testing.T) {
	t.Parallel()

	if got := (config.Interop{}).TransportFor("user_registration"); got != "inproc" {
		t.Errorf("default transport = %q, want \"inproc\"", got)
	}
}
