package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestMissingDriverFallsBackToZeroDependencyDefault: a module enabled without
// an explicit driver takes the dependency-free driver, never the most complete
// one.
func TestMissingDriverFallsBackToZeroDependencyDefault(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"outbox":      "memory",
		"idempotency": "memory",
		"dynconf":     "file",
		"audit":       "log",
		"storage":     "disk",
		"scheduler":   "cron-inproc",
	}

	for module, want := range cases {
		t.Run(module, func(t *testing.T) {
			t.Parallel()
			modules := config.Modules{module: {Enabled: true}}
			if got := modules.Resolve(shippedCatalog(t)).DriverOf(module); got != want {
				t.Errorf("default driver of %s = %q, want %q", module, got, want)
			}
		})
	}
}
