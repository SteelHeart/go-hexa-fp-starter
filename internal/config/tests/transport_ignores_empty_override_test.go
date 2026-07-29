package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestTransportIgnoresEmptyOverride: an empty entry in the configuration must
// not silently disable the transport — it falls back on the default.
func TestTransportIgnoresEmptyOverride(t *testing.T) {
	t.Parallel()

	interop := config.Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"billing": ""},
	}
	if got := interop.TransportFor("billing"); got != "inproc" {
		t.Errorf("empty override = %q, want the default \"inproc\"", got)
	}
}
