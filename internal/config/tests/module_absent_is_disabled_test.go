package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestModuleAbsentIsDisabled: deny by default, all the way into the
// configuration. One never enables a capability that nobody asked for.
func TestModuleAbsentIsDisabled(t *testing.T) {
	t.Parallel()

	// Resolve is the step that places the defaults, and it is EXPLICIT since
	// ADR 014: the catalogue comes from the composition root, not from a hidden
	// table.
	got := config.Modules{}.Resolve(shippedCatalog(t)).Get("outbox")
	if got.Enabled {
		t.Error("a module absent from the configuration must be disabled")
	}
	if got.Driver != "memory" {
		t.Errorf("default driver = %q, want \"memory\" (without an external dependency)", got.Driver)
	}
}
