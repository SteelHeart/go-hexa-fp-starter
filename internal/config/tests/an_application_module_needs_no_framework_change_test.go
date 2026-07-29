package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestAnApplicationModuleNeedsNoFrameworkChange is the test that DEMONSTRATES
// ADR 014. Without it, the decision would remain an intention.
//
// Before, this was impossible: the only module table lived in
// `internal/config/modules.go`, and the binary answered
// `modules.facturation: unknown module` with a return code of 1. Declaring
// one's own module forced one to modify a file of the framework.
//
// Here, `facturation` exists nowhere in the starter — no code, no driver, no
// line in `internal/config`. It only has a catalogue, such as an application
// would supply. And the configuration accepts it.
//
// To be read with `TestAModuleAbsentFromTheCatalogIsRefused`: one proves that
// one can add, the other that one cannot add just anything.
func TestAnApplicationModuleNeedsNoFrameworkChange(t *testing.T) {
	withCatalogTestConfig(t, "modules:\n  facturation:\n    enabled: true\n    driver: sqlite\n")

	catalog := applicationCatalog()
	cfg, err := config.Load(catalog)
	if err != nil {
		t.Fatalf("an application module must be accepted without touching the starter: %v", err)
	}
	if got := cfg.Modules.DriverOf("facturation"); got != "sqlite" {
		t.Errorf("driver retained = %q, want \"sqlite\"", got)
	}
	if !cfg.Modules.RequiresSQL(catalog) {
		t.Error("the sqlite driver declares that it requires a database: RequiresSQL must see it")
	}
}
