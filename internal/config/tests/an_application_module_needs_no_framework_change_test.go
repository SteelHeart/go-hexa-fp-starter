package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestAnApplicationModuleNeedsNoFrameworkChange est le test qui DÉMONTRE
// l'ADR 014. Sans lui, la décision resterait une intention.
//
// Avant, ceci était impossible : la seule table de modules vivait dans
// `internal/config/modules.go`, et le binaire répondait
// `modules.facturation : module inconnu` avec un code de retour 1. Déclarer son
// propre module obligeait à modifier un fichier du framework.
//
// Ici, `facturation` n'existe nulle part dans le socle — ni code, ni pilote, ni
// ligne dans `internal/config`. Il n'a qu'un catalogue, tel qu'une application
// en fournirait un. Et la configuration l'accepte.
//
// À lire avec `TestAModuleAbsentFromTheCatalogIsRefused` : l'un prouve qu'on
// peut ajouter, l'autre qu'on ne peut pas ajouter n'importe quoi.
func TestAnApplicationModuleNeedsNoFrameworkChange(t *testing.T) {
	withCatalogTestConfig(t, "modules:\n  facturation:\n    enabled: true\n    driver: sqlite\n")

	catalog := applicationCatalog()
	cfg, err := config.Load(catalog)
	if err != nil {
		t.Fatalf("un module applicatif doit être accepté sans toucher au socle: %v", err)
	}
	if got := cfg.Modules.DriverOf("facturation"); got != "sqlite" {
		t.Errorf("pilote retenu = %q, attendu \"sqlite\"", got)
	}
	if !cfg.Modules.RequiresSQL(catalog) {
		t.Error("le pilote sqlite déclare exiger une base : RequiresSQL doit le voir")
	}
}
