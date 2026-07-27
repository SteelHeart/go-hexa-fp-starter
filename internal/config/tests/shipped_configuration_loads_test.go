package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedConfigurationLoads : la configuration livrée doit charger telle
// quelle, le seul secret obligatoire étant fourni.
func TestShippedConfigurationLoads(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("la configuration livrée doit charger: %v", err)
	}
	if len(cfg.Modules) == 0 {
		t.Error("aucun module lu : le fichier modules.yaml n'a pas été pris en compte")
	}
}
