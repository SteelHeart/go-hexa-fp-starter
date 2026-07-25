package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler/application"
)

// TestDisabledModuleRefusesLoudly : un ordonnanceur désactivé refuse à l'appel.
//
// `Acquire` rend `false` ET une erreur, et les deux comptent : `false` garantit
// qu'aucune tâche ne s'exécute, l'erreur garantit que le refus se voit. Un
// ordonnanceur silencieusement inerte est un défaut qu'on découvre en constatant
// que le travail n'a pas été fait depuis des semaines.
func TestDisabledModuleRefusesLoudly(t *testing.T) {
	t.Parallel()

	mod, err := scheduler.New(
		config.Module{Enabled: false, Driver: "cron-inproc"},
		scheduler.Deps{Logger: discardLogger()},
	)
	if err != nil {
		t.Fatalf("un module désactivé se construit sans erreur: %v", err)
	}
	ctx := context.Background()

	if err := mod.Run(ctx, []application.Scheduled{}); !errors.Is(err, scheduler.ErrDisabled) {
		t.Errorf("Run = %v, attendu ErrDisabled", err)
	}

	elected, err := mod.Acquire(ctx, "purge")
	if !errors.Is(err, scheduler.ErrDisabled) {
		t.Errorf("Acquire = %v, attendu ErrDisabled", err)
	}
	if elected {
		t.Error("un module désactivé ne doit élire personne")
	}

	if err := mod.Release(ctx, "purge"); !errors.Is(err, scheduler.ErrDisabled) {
		t.Errorf("Release = %v, attendu ErrDisabled", err)
	}
}
