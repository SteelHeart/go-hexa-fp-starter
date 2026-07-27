package tests

import (
	"testing"

	"github.com/SteelHeart/go-hexa-fp-starter/internal/config"
	"github.com/SteelHeart/go-hexa-fp-starter/internal/core/scheduler"
)

// TestUnknownDriverRefusesStartup : deny par défaut jusque dans la fabrique.
//
// `external` figure dans le catalogue d'intentions mais n'est PAS construit : il
// doit donc refuser franchement, jamais se replier sur `cron-inproc` — qui
// exécuterait la tâche sur chaque réplique.
func TestUnknownDriverRefusesStartup(t *testing.T) {
	t.Parallel()

	for _, driver := range []string{"external", "cron", "n'importe quoi"} {
		if _, err := scheduler.New(
			config.Module{Enabled: true, Driver: driver},
			scheduler.Deps{Logger: discardLogger()},
		); err == nil {
			t.Errorf("le pilote %q n'est pas construit : il doit refuser le démarrage", driver)
		}
	}
}
