package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedConfigurationNeedsNoInfrastructure: with the shipped drivers, no
// external service is required. This test fails the day somebody enables a
// postgres or redis driver by default — and that is precisely the point.
func TestShippedConfigurationNeedsNoInfrastructure(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if cfg.Modules.RequiresSQL(shippedCatalog(t)) {
		t.Error(`the shipped configuration requires an SQL database: the "zero prerequisite" promise is broken`)
	}
	if cfg.Modules.RequiresCache(shippedCatalog(t)) {
		t.Error(`the shipped configuration requires a cache: the "zero prerequisite" promise is broken`)
	}
}
