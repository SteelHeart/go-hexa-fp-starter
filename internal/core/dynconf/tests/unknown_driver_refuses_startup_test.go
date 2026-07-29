package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/dynconf"
)

// TestUnknownDriverRefusesStartup: deny by default, right down to the factory.
//
// Configuration validation has already rejected the unknown driver; this second
// refusal guarantees that no path — a caller that builds the module by hand,
// for instance — bypasses the first.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := dynconf.New(config.Module{Enabled: true, Driver: "consul"}, dynconf.Deps{})
	if err == nil {
		t.Fatal("an unknown driver must refuse to start")
	}
}
