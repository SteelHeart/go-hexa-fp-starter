package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestAModuleAbsentFromTheCatalogIsRefused : le deny-par-défaut n'a pas faibli
// en changeant de source.
//
// Ouvrir le schéma aurait été la solution facile — accepter n'importe quel nom
// et laisser chaque module se valider lui-même. `factration` au lieu de
// `facturation` serait alors accepté, et le module resterait silencieusement
// désactivé : le mode de défaillance que ce dépôt combat le plus, celui qui ne
// se signale jamais.
func TestAModuleAbsentFromTheCatalogIsRefused(t *testing.T) {
	withCatalogTestConfig(t, "modules:\n  factration:\n    enabled: true\n")

	_, err := config.Load(applicationCatalog())
	if err == nil {
		t.Fatal("un module absent du catalogue doit refuser le démarrage")
	}
	if !strings.Contains(err.Error(), "factration") {
		t.Errorf("le message doit nommer le module fautif: %v", err)
	}
}
