package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
)

// TestAdvisoryLockDriverRefusesWithoutPool: the driver that requires a database
// refuses at startup.
//
// This test also locks down the fix for a design defect of the previous
// version: it required PostgreSQL SIMPLY to repeat a task, including in a
// single-instance binary which has nobody to agree with. From now on the
// election is a driver, and the default driver requires nothing — the database
// is only needed if it is chosen.
func TestAdvisoryLockDriverRefusesWithoutPool(t *testing.T) {
	t.Parallel()

	_, err := scheduler.New(
		config.Module{Enabled: true, Driver: "advisory-lock"},
		scheduler.Deps{Logger: discardLogger()},
	)
	if !errors.Is(err, scheduler.ErrPoolRequired) {
		t.Errorf("error = %v, want ErrPoolRequired", err)
	}
}
