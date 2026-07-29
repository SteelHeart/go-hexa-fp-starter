package tests

import (
	"slices"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/core"
)

// TestEveryModuleDefaultDriverIsDeclared: the default driver of a module must
// appear among its allowed drivers.
//
// A default missing from the list would make a correctly declared module refuse
// to start, with a message blaming the user for a fault that is ours.
//
// An EMPTY default would produce the same refusal, but worse: the message would
// quote a driver `""` that nobody ever wrote.
func TestEveryModuleDefaultDriverIsDeclared(t *testing.T) {
	t.Parallel()

	catalog, err := core.Catalog()
	if err != nil {
		t.Fatalf("core catalogue: %v", err)
	}
	if len(catalog) == 0 {
		t.Fatal("empty catalogue: this test would check nothing")
	}

	for name := range catalog {
		allowed := catalog.AllowedDrivers(name)
		fallback := catalog.DefaultDriver(name)
		if fallback == "" {
			t.Errorf("module %q: no default driver", name)
			continue
		}
		if !slices.Contains(allowed, fallback) {
			t.Errorf("module %q: default %q missing from allowed drivers %v", name, fallback, allowed)
		}
	}
}
