package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestTransportSwitchIsPureConfiguration documents the intention of ADR 010:
// moving a module from inproc to http means extracting it into a service
// without touching a line of code. This test would fail if the resolution
// became implicit.
func TestTransportSwitchIsPureConfiguration(t *testing.T) {
	t.Parallel()

	before := config.Interop{DefaultTransport: "inproc"}
	after := config.Interop{
		DefaultTransport: "inproc",
		Transports:       map[string]string{"user_registration": "http"},
		BaseURLs:         map[string]string{"user_registration": "http://user-registration:8080"},
	}

	if before.TransportFor("user_registration") != "inproc" {
		t.Fatal("unexpected initial state")
	}
	if after.TransportFor("user_registration") != "http" {
		t.Error("the change of mode must come from the configuration alone")
	}
}
