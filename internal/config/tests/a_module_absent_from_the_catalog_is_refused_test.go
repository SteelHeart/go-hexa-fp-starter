package tests

import (
	"strings"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestAModuleAbsentFromTheCatalogIsRefused: deny-by-default has not weakened by
// changing source.
//
// Opening the schema would have been the easy solution — accept any name and
// let each module validate itself. `factration` instead of `facturation` would
// then be accepted, and the module would stay silently disabled: the failure
// mode this repository fights the most, the one that never reports itself.
func TestAModuleAbsentFromTheCatalogIsRefused(t *testing.T) {
	withCatalogTestConfig(t, "modules:\n  factration:\n    enabled: true\n")

	_, err := config.Load(applicationCatalog())
	if err == nil {
		t.Fatal("a module absent from the catalogue must refuse to start")
	}
	if !strings.Contains(err.Error(), "factration") {
		t.Errorf("the message must name the offending module: %v", err)
	}
}
