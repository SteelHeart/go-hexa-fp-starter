package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/audit"
)

// TestUnknownDriverRefusesStartup: deny by default, right down to the factory.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	_, err := audit.New(config.Module{Enabled: true, Driver: "syslog"}, audit.Deps{})
	if err == nil {
		t.Fatal("an unknown driver must refuse to start")
	}
}
