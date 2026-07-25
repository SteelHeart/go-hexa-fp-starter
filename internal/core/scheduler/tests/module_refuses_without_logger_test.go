package tests

import (
	"errors"
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
)

// TestModuleRefusesWithoutLogger : le journal n'est pas un confort.
//
// L'orchestration ne journalise pas — elle rend compte (ports.Report) — mais le
// compte rendu doit bien aboutir quelque part. Sans destination, une tâche en échec
// échouerait en silence : c'est le pire défaut possible pour un travail qui tourne
// sans témoin, la nuit, sans personne pour constater qu'il n'a rien fait.
func TestModuleRefusesWithoutLogger(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"cron-inproc", "advisory-lock"} {
		_, err := scheduler.New(
			config.Module{Enabled: true, Driver: driver},
			scheduler.Deps{},
		)
		if !errors.Is(err, scheduler.ErrLoggerRequired) {
			t.Errorf("pilote %q: erreur = %v, attendu ErrLoggerRequired", driver, err)
		}
	}
}
