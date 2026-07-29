package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
)

// TestDisabledModuleRefusesLoudly: a disabled scheduler refuses when called.
//
// `Acquire` returns `false` AND an error, and both matter: `false` guarantees
// that no task runs, the error guarantees that the refusal is visible. A
// silently inert scheduler is a defect one discovers upon noticing that the
// work has not been done for weeks.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := scheduler.New(
		config.Module{Enabled: false, Driver: "cron-inproc"},
		scheduler.Deps{Logger: discardLogger()},
	)
	if err != nil {
		t.Fatalf("a disabled module builds without error: %v", err)
	}
	ctx := context.Background()

	if runErr := mod.Run(ctx, []application.Scheduled{}); !errors.Is(runErr, scheduler.ErrDisabled) {
		t.Errorf("Run = %v, want ErrDisabled", runErr)
	}

	elected, err := mod.Acquire(ctx, "purge")
	if !errors.Is(err, scheduler.ErrDisabled) {
		t.Errorf("Acquire = %v, want ErrDisabled", err)
	}
	if elected {
		t.Error("a disabled module must elect nobody")
	}

	if releaseErr := mod.Release(ctx, "purge"); !errors.Is(releaseErr, scheduler.ErrDisabled) {
		t.Errorf("Release = %v, want ErrDisabled", releaseErr)
	}
}
