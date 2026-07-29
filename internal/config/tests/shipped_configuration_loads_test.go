package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedConfigurationLoads: the shipped configuration must load as it is,
// the only mandatory secret being supplied.
func TestShippedConfigurationLoads(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("the shipped configuration must load: %v", err)
	}
	if len(cfg.Modules) == 0 {
		t.Error("no module read: the modules.yaml file was not taken into account")
	}
}
