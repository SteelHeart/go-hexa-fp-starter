package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
)

// TestUnknownDriverRefusesStartup: deny by default, right down to the factory.
//
// `external` appears in the catalogue of intentions but is NOT built: it must
// therefore refuse outright, never fall back on `cron-inproc` — which would run
// the task on every replica.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"external", "cron", "anything at all"} {
		if _, err := scheduler.New(
			config.Module{Enabled: true, Driver: driver},
			scheduler.Deps{Logger: discardLogger()},
		); err == nil {
			t.Errorf("the %q driver is not built: it must refuse to start", driver)
		}
	}
}
