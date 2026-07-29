package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
)

// TestModuleRefusesWithoutLogger: the logger is not a comfort.
//
// The orchestration does not log — it reports (ports.Report) — but the report
// has to end up somewhere. With no destination, a failing task would fail in
// silence: that is the worst possible defect for work that runs with no
// witness, at night, with nobody to notice that it did nothing.
func TestModuleRefusesWithoutLogger(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"cron-inproc", "advisory-lock"} {
		_, err := scheduler.New(
			config.Module{Enabled: true, Driver: driver},
			scheduler.Deps{},
		)
		if !errors.Is(err, scheduler.ErrLoggerRequired) {
			t.Errorf("driver %q: error = %v, want ErrLoggerRequired", driver, err)
		}
	}
}
