package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestShippedConfigurationNeedsNoInfrastructure : avec les pilotes livrés, aucun
// service externe n'est requis. Ce test échoue le jour où quelqu'un active un
// pilote postgres ou redis par défaut — et c'est précisément le but.
func TestShippedConfigurationNeedsNoInfrastructure(t *testing.T) {
	withShippedConfig(t)

	cfg, err := config.Load(shippedCatalog(t))
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}
	if cfg.Modules.RequiresSQL(shippedCatalog(t)) {
		t.Error("la configuration livrée exige une base SQL : la promesse « zéro prérequis » est rompue")
	}
	if cfg.Modules.RequiresCache(shippedCatalog(t)) {
		t.Error("la configuration livrée exige un cache : la promesse « zéro prérequis » est rompue")
	}
}
