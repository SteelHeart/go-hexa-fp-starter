package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
)

// TestMissingDriverFallsBackToZeroDependencyDefault : un module activé sans
// pilote explicite prend le pilote sans dépendance, jamais le plus complet.
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
			if got := modules.DriverOf(module); got != want {
				t.Errorf("pilote par défaut de %s = %q, attendu %q", module, got, want)
			}
		})
	}
}
