package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

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
			t.Errorf("transport of %s = %q, want %q", module, got, want)
		}
	}
}
